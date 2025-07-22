package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/MdSadiqMd/snipzy-worker/internal/dto"
	"github.com/MdSadiqMd/snipzy-worker/pkg/config"
)

type YTDLPService struct {
	config *config.Config
}

func NewYTDLPService(cfg *config.Config) *YTDLPService {
	return &YTDLPService{config: cfg}
}

func (s *YTDLPService) GetVideoInfo(ctx context.Context, videoURL string) (*dto.VideoInfo, error) {
	cmd := exec.CommandContext(ctx, s.config.YTDLPPath,
		"-g",
		"-j",
		"--no-playlist",
		"--no-warnings",
		videoURL)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("yt-dlp failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("invalid yt-dlp output")
	}

	var info dto.VideoInfo
	if err := json.Unmarshal([]byte(lines[0]), &info); err != nil {
		return nil, fmt.Errorf("failed to parse video info: %w", err)
	}

	info.DirectURL = lines[1]
	return &info, nil
}

func (s *YTDLPService) GetVideoURLWithTimeRange(ctx context.Context, videoURL string, startTime, duration float64) (string, error) {
	cmd := exec.CommandContext(ctx, s.config.YTDLPPath,
		"-g",
		"--download-sections", fmt.Sprintf("*%f-%f", startTime, startTime+duration),
		"--no-playlist",
		"--no-warnings",
		videoURL)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("yt-dlp range request failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

func (s *YTDLPService) CalculateByteRange(info *dto.VideoInfo, startTime, duration float64) (start, end int64) {
	if info.Duration <= 0 {
		return 0, 1024 * 1024
	}

	totalDuration := info.Duration
	estimatedFileSize := int64(50 * 1024 * 1024)
	for _, format := range info.Formats {
		if format.Filesize > 0 {
			estimatedFileSize = format.Filesize
			break
		}
	}

	bytesPerSecond := float64(estimatedFileSize) / totalDuration
	start = int64(startTime * bytesPerSecond)
	end = int64((startTime + duration) * bytesPerSecond)
	if end > estimatedFileSize {
		end = estimatedFileSize - 1
	}

	return start, end
}
