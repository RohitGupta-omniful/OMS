package handlers

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Handler struct {
	S3Client *s3.Client
}

func NewHandler(s3Client *s3.Client) *Handler {
	if s3Client == nil {
		fmt.Println("S3 client is not set up")
	} else {
		fmt.Println("S3 client successfully set up")
	}
	return &Handler{
		S3Client: s3Client,
	}
}
