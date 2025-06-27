package handlers

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/omniful/go_commons/i18n"
	"github.com/omniful/go_commons/log"
)

type Handler struct {
	S3Client *s3.Client
}

func NewHandler(ctx context.Context, s3Client *s3.Client) *Handler {
	if s3Client == nil {
		log.Warnf(i18n.Translate(ctx, "S3 client is not set up"))
	} else {
		log.Infof(i18n.Translate(ctx, "S3 client successfully set up"))
	}
	return &Handler{
		S3Client: s3Client,
	}
}
