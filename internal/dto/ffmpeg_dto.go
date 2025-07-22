package dto

import (
	"io"
	"os/exec"
)

type ProcessingOptions struct {
	StartTime    float64
	Duration     float64
	Platform     string
	Quality      string
	OutputFormat string
}

type PlatformPreset struct {
	Width        int
	Height       int
	FilterString string
}

type ProcessReader struct {
	Reader io.ReadCloser
	Cmd    *exec.Cmd
}

func (pr *ProcessReader) Read(p []byte) (n int, err error) {
	return pr.Reader.Read(p)
}

func (pr *ProcessReader) Close() error {
	pr.Reader.Close()
	return pr.Cmd.Wait()
}
