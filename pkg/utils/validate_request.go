package utils

import (
	"fmt"

	"github.com/MdSadiqMd/snipzy-worker/internal/dto"
	"github.com/MdSadiqMd/snipzy-worker/pkg/config"
)

func ValidateRequest(req *dto.ClipRequest) error {
	if req.URL == "" {
		return fmt.Errorf("URL is required")
	}
	if req.Start < 0 {
		return fmt.Errorf("start time must be non-negative")
	}
	if req.End <= req.Start {
		return fmt.Errorf("end time must be greater than start time")
	}

	duration := req.End - req.Start
	if duration > float64(config.New().MaxDuration) {
		return fmt.Errorf("duration exceeds maximum of %d seconds", config.New().MaxDuration)
	}

	if req.Platform == "" {
		req.Platform = "default"
	}
	if req.Quality == "" {
		req.Quality = "medium"
	}
	return nil
}
