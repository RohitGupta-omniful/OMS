package IMS_APIS

import (
	"context"
	"errors"

	"github.com/omniful/go_commons/config"
	"github.com/omniful/go_commons/http"
	"github.com/omniful/go_commons/i18n"
	interservice_client "github.com/omniful/go_commons/interservice-client"
	"github.com/omniful/go_commons/log"
)

var imsClient *interservice_client.Client
var authToken string

func InitIMSClient(ctx context.Context) error {
	configIMS := interservice_client.Config{
		ServiceName: config.GetString(ctx, "interservice_client.serviceName"),
		BaseURL:     config.GetString(ctx, "interservice_client.baseURL"),
		Timeout:     config.GetDuration(ctx, "interservice_client.timeout"),
	}

	client, err := interservice_client.NewClientWithConfig(configIMS)
	if err != nil {
		return err
	}

	authToken = "my-secret-token"
	if authToken == "" {
		return errors.New(i18n.Translate(ctx, "IMS auth token is not set"))
	}

	log.Infof(i18n.Translate(ctx, "Connected to INTER_SERVICE Client"))
	SetIMSClient(client)
	return nil
}

func SetIMSClient(client *interservice_client.Client) {
	imsClient = client
}

type ValidationResponse struct {
	IsValid bool `json:"is_valid"`
}

func ValidateHub(ctx context.Context, hubID string) bool {
	if imsClient == nil {
		log.Error(i18n.Translate(ctx, "IMS client is not initialized"))
		return false
	}

	req := &http.Request{
		Url:     "/validate/hub/" + hubID,
		Headers: map[string][]string{"Authorization": {"Bearer " + authToken}},
	}

	var result ValidationResponse
	_, err := imsClient.Get(req, &result)
	if err != nil {
		log.Errorf(i18n.Translate(ctx, "Error validating hubID %s: %v"), hubID, err)
		return false
	}

	return result.IsValid
}

func ValidateSKUOnHub(ctx context.Context, skuID string) bool {
	if imsClient == nil {
		log.Error(i18n.Translate(ctx, "IMS client is not initialized"))
		return false
	}

	req := &http.Request{
		Url:     "/validate/sku/" + skuID,
		Headers: map[string][]string{"Authorization": {"Bearer " + authToken}},
	}

	var result ValidationResponse
	_, err := imsClient.Get(req, &result)
	if err != nil {
		log.Errorf(i18n.Translate(ctx, "Error validating SKU %s: %v"), skuID, err)
		return false
	}

	return result.IsValid
}
