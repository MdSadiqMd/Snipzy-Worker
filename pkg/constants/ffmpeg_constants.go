package constants

import (
	"strconv"

	"github.com/MdSadiqMd/snipzy-worker/internal/dto"
)

func GetPlatformPresets() map[string]dto.PlatformPreset {
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

func BuildFFmpegArgs(inputFile string, options dto.ProcessingOptions, preset dto.PlatformPreset) []string {
	args := []string{
		"-ss", strconv.FormatFloat(options.StartTime, 'f', 2, 64),
		"-i", inputFile,
		"-t", strconv.FormatFloat(options.Duration, 'f', 2, 64),
		"-filter_complex", preset.FilterString,
		"-map", "[v]",
		"-map", "[a]",
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", GetCRF(options.Quality),
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		"-f", "mp4",
		"pipe:1",
	}
	return args
}

func GetCRF(quality string) string {
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
