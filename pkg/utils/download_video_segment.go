package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/MdSadiqMd/snipzy-worker/internal/server"
)

func DownloadVideoSegment(ctx context.Context, url, outputPath string, startTime, duration float64) error {
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
