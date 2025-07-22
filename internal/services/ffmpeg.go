package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/MdSadiqMd/snipzy-worker/internal/dto"
	"github.com/MdSadiqMd/snipzy-worker/internal/server"
	"github.com/MdSadiqMd/snipzy-worker/pkg/config"
)

type FFmpegService struct {
	config *config.Config
}

func NewFFmpegService(cfg *config.Config) *FFmpegService {
	return &FFmpegService{config: cfg}
}

func (s *FFmpegService) GetPlatformPresets() map[string]dto.PlatformPreset {
	return map[string]dto.PlatformPreset{
		"instagram-feed": {
			Width:        1080,
			Height:       1080,
			FilterString: "[0:v]scale=1080:1080:force_original_aspect_ratio=decrease,pad=1080:1080:(ow-iw)/2:(oh-ih)/2[v];[1:a]aresample=48000,volume=1[a]",
		},
		"instagram-story": {
			Width:        1080,
			Height:       1920,
			FilterString: "[0:v]scale=1080:1920:force_original_aspect_ratio=decrease,pad=1080:1920:(ow-iw)/2:(oh-ih)/2[v];[1:a]aresample=48000,volume=1[a]",
		},
		"youtube": {
			Width:        1920,
			Height:       1080,
			FilterString: "[0:v]scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2[v];[1:a]aresample=48000,volume=1[a]",
		},
		"twitter": {
			Width:        1280,
			Height:       720,
			FilterString: "[0:v]scale=1280:720:force_original_aspect_ratio=decrease,pad=1280:720:(ow-iw)/2:(oh-ih)/2[v];[1:a]aresample=48000,volume=1[a]",
		},
		"default": {
			Width:        1280,
			Height:       720,
			FilterString: "[0:v]scale=1280:720[v];[1:a]aresample=48000,volume=1[a]",
		},
	}
}

func (s *FFmpegService) ProcessVideoStream(ctx context.Context, directURL string, options dto.ProcessingOptions) (io.ReadCloser, error) {
	preset, exists := s.GetPlatformPresets()[options.Platform]
	if !exists {
		preset = s.GetPlatformPresets()["default"]
	}

	inputFile := filepath.Join(s.config.TmpDir, "input_segment.mp4")
	if err := s.downloadVideoSegment(ctx, directURL, inputFile, options.StartTime, options.Duration); err != nil {
		return nil, fmt.Errorf("failed to download segment: %w", err)
	}
	defer os.Remove(inputFile)

	args := s.buildFFmpegArgs(inputFile, options, preset)
	cmd := exec.CommandContext(ctx, s.config.FFmpegPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	return &dto.ProcessReader{
		Reader: stdout,
		Cmd:    cmd,
	}, nil
}

func (s *FFmpegService) downloadVideoSegment(ctx context.Context, url, outputPath string, startTime, duration float64) error {
	supportsRange, totalSize, err := server.CheckRangeSupport(url)
	if err != nil {
		return fmt.Errorf("failed to check range support: %w", err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	if supportsRange && totalSize > 0 {
		estimatedBytesPerSecond := float64(totalSize) / 3600
		start := int64(startTime * estimatedBytesPerSecond)
		end := int64((startTime + duration) * estimatedBytesPerSecond)
		if end >= totalSize {
			end = totalSize - 1
		}

		resp, err := server.HTTPRangeRequest(url, start, end)
		if err != nil {
			return fmt.Errorf("range request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusPartialContent {
			return fmt.Errorf("server returned status %d instead of 206", resp.StatusCode)
		}

		_, err = io.Copy(outFile, resp.Body)
		return err
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(outFile, resp.Body)
	return err
}

func (s *FFmpegService) buildFFmpegArgs(inputFile string, options dto.ProcessingOptions, preset dto.PlatformPreset) []string {
	args := []string{
		"-ss", strconv.FormatFloat(options.StartTime, 'f', 2, 64),
		"-i", inputFile,
		"-t", strconv.FormatFloat(options.Duration, 'f', 2, 64),
		"-filter_complex", preset.FilterString,
		"-map", "[v]",
		"-map", "[a]",
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", s.getCRF(options.Quality),
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-f", "mp4",
		"pipe:1",
	}
	return args
}

func (s *FFmpegService) getCRF(quality string) string {
	switch quality {
	case "high":
		return "18"
	case "medium":
		return "23"
	case "low":
		return "28"
	default:
		return "23"
	}
}
