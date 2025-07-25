package main

import (
	"github.com/MdSadiqMd/snipzy-worker/internal/handlers"
	"github.com/MdSadiqMd/snipzy-worker/pkg/config"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	cfg := config.New()
	handler := handlers.NewVideoHandler(cfg)
	lambda.Start(handler.HandleClipRequest)
}
