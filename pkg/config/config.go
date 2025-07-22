package config

import (
	"os"
	"strconv"
)

type Config struct {
	FFmpegPath    string
	YTDLPPath     string
	MemoryLimitMB int
	TmpDir        string
	MaxDuration   int // Maximum clip duration in seconds
}

func New() *Config {
	memoryLimit, _ := strconv.Atoi(getEnv("AWS_LAMBDA_FUNCTION_MEMORY_SIZE", "2048"))
	maxDuration, _ := strconv.Atoi(getEnv("MAX_CLIP_DURATION", "30"))

	return &Config{
		FFmpegPath:    getEnv("FFMPEG_PATH", "/opt/bin/ffmpeg"),
		YTDLPPath:     getEnv("YTDLP_PATH", "/opt/bin/yt-dlp"),
		MemoryLimitMB: memoryLimit,
		TmpDir:        "/tmp",
		MaxDuration:   maxDuration,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
