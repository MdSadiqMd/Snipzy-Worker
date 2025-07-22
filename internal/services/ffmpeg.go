package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/MdSadiqMd/snipzy-worker/internal/dto"
	"github.com/MdSadiqMd/snipzy-worker/pkg/config"
	"github.com/MdSadiqMd/snipzy-worker/pkg/constants"
	"github.com/MdSadiqMd/snipzy-worker/pkg/utils"
)

type FFmpegService struct {
	config *config.Config
}

func NewFFmpegService(cfg *config.Config) *FFmpegService {
	return &FFmpegService{config: cfg}
}

func (s *FFmpegService) ProcessVideoStream(ctx context.Context, directURL string, options dto.ProcessingOptions) (io.ReadCloser, error) {
	preset, exists := constants.GetPlatformPresets()[options.Platform]
	if !exists {
		preset = constants.GetPlatformPresets()["default"]
	}

	inputFile := filepath.Join(s.config.TmpDir, "input_segment.mp4")
	if err := utils.DownloadVideoSegment(ctx, directURL, inputFile, options.StartTime, options.Duration); err != nil {
		return nil, fmt.Errorf("failed to download segment: %w", err)
	}
	defer os.Remove(inputFile)

	args := constants.BuildFFmpegArgs(inputFile, options, preset)
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
