package dto

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
