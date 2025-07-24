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
