package res

import (
	"encoding/json"
	"strings"

	"github.com/MdSadiqMd/snipzy-worker/internal/dto"
	"github.com/aws/aws-lambda-go/events"
)

func ErrorResponse(statusCode int, message string) *events.LambdaFunctionURLStreamingResponse {
	response := dto.ClipResponse{Error: message}
	body, _ := json.Marshal(response)

	return &events.LambdaFunctionURLStreamingResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: strings.NewReader(string(body)),
	}
}
