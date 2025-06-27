package webkooks

import (
	"context"

	"github.com/omniful/go_commons/httpclient"
	"github.com/omniful/go_commons/httpclient/request"
	"github.com/omniful/go_commons/i18n"
	"github.com/omniful/go_commons/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var WebhookCollection *mongo.Collection

type Webhook struct {
	URL      string `json:"url" bson:"url"`
	TenantID int64  `json:"tenant_id" bson:"tenant_id"`
}

func SetWebhookCollection(col *mongo.Collection) {
	WebhookCollection = col
}

func NotifyTenantWebhook(ctx context.Context, tenantID int64, payload interface{}) {
	log.Infof(i18n.Translate(ctx, "Preparing to notify tenant webhook for TenantID=%d"), tenantID)

	if WebhookCollection == nil {
		log.Error(i18n.Translate(ctx, "WebhookCollection is not initialized"))
		return
	}

	var wh Webhook
	err := WebhookCollection.FindOne(ctx, bson.M{"tenant_id": tenantID}).Decode(&wh)
	if err != nil {
		log.Warnf(i18n.Translate(ctx, "Failed to retrieve webhook for TenantID=%d: %v"), tenantID, err)
		return
	}

	urlStr := wh.URL
	client := httpclient.New("")

	req, err := request.NewBuilder().
		SetMethod("POST").
		SetUri(urlStr).
		SetHeaders(map[string][]string{
			"Content-Type": {"application/json"},
		}).
		SetBody(payload).
		Build()
	if err != nil {
		log.Error(i18n.Translate(ctx, "Failed to build webhook request"), err)
		return
	}

	log.Infof(i18n.Translate(ctx, "Sending webhook to TenantID=%d at URL=%s"), tenantID, urlStr)

	resp, err := client.Post(ctx, req)
	if err != nil {
		log.Error(i18n.Translate(ctx, "Failed to send webhook to ")+urlStr, err)
		return
	}

	log.Infof(i18n.Translate(ctx, "Successfully sent webhook for TenantID=%d to URL=%s. Status: %d"), tenantID, urlStr, resp.StatusCode)
}
