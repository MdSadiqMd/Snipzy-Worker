package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MdSadiqMd/snipzy-worker/internal/dto"
	"github.com/MdSadiqMd/snipzy-worker/internal/services"
	"github.com/MdSadiqMd/snipzy-worker/pkg/config"
	"github.com/MdSadiqMd/snipzy-worker/pkg/res"
	"github.com/MdSadiqMd/snipzy-worker/pkg/utils"
	"github.com/aws/aws-lambda-go/events"
)

type VideoHandler struct {
	config        *config.Config
	ytdlpService  *services.YTDLPService
	ffmpegService *services.FFmpegService
}

func NewVideoHandler(cfg *config.Config) *VideoHandler {
	return &VideoHandler{
		config:        cfg,
		ytdlpService:  services.NewYTDLPService(cfg),
		ffmpegService: services.NewFFmpegService(cfg),
	}
}

func (h *VideoHandler) HandleClipRequest(ctx context.Context, request events.APIGatewayProxyRequest) (*events.LambdaFunctionURLStreamingResponse, error) {
	var clipReq dto.ClipRequest
	if err := json.Unmarshal([]byte(request.Body), &clipReq); err != nil {
		return res.ErrorResponse(http.StatusBadRequest, "Invalid request body"), nil
	}
	if err := utils.ValidateRequest(&clipReq); err != nil {
		return res.ErrorResponse(http.StatusBadRequest, err.Error()), nil
	}

	processCtx, cancel := context.WithTimeout(ctx, 12*time.Minute) // Leave 3 minutes buffer
	defer cancel()

	videoInfo, err := h.ytdlpService.GetVideoInfo(processCtx, clipReq.URL)
	if err != nil {
		return res.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Failed to get video info: %v", err)), nil
	}

	duration := clipReq.End - clipReq.Start
	if duration > float64(h.config.MaxDuration) {
		return res.ErrorResponse(http.StatusBadRequest, fmt.Sprintf("Duration exceeds maximum of %d seconds", h.config.MaxDuration)), nil
	}

	options := dto.ProcessingOptions{
		StartTime:    clipReq.Start,
		Duration:     duration,
		Platform:     clipReq.Platform,
		Quality:      clipReq.Quality,
		OutputFormat: "mp4",
	}

	videoStream, err := h.ffmpegService.ProcessVideoStream(processCtx, videoInfo.DirectURL, options)
	if err != nil {
		return res.ErrorResponse(http.StatusInternalServerError, fmt.Sprintf("Video processing failed: %v", err)), nil
	}

	return &events.LambdaFunctionURLStreamingResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":        "video/mp4",
			"Content-Disposition": `attachment; filename="clip.mp4"`,
			"Cache-Control":       "no-cache",
		},
		Body: videoStream,
	}, nil
}
