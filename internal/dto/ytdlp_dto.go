package dto

type VideoInfo struct {
	URL       string   `json:"url"`
	Title     string   `json:"title"`
	Duration  float64  `json:"duration"`
	Formats   []Format `json:"formats"`
	DirectURL string   `json:"direct_url,omitempty"`
}

type Format struct {
	FormatID   string `json:"format_id"`
	URL        string `json:"url"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	Filesize   int64  `json:"filesize,omitempty"`
	VideoCodec string `json:"vcodec,omitempty"`
	AudioCodec string `json:"acodec,omitempty"`
}
