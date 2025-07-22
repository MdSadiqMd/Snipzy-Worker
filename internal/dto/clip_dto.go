package dto

type ClipRequest struct {
	URL      string  `json:"url"`
	Start    float64 `json:"start"`
	End      float64 `json:"end"`
	Platform string  `json:"platform"`
	Quality  string  `json:"quality"`
}

type ClipResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}
