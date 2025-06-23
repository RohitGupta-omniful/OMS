package IMS_APIS

import (
	"context"
	"fmt"
	"log"

	"github.com/omniful/go_commons/config"
	"github.com/omniful/go_commons/http"
	interservice_client "github.com/omniful/go_commons/interservice-client"
)

var imsClient *interservice_client.Client

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
	log.Println("Connected to INTER_SERVICE Client")
	SetIMSClient(client)
	return nil
}

func SetIMSClient(client *interservice_client.Client) {
	imsClient = client
}

type ValidationResponse struct {
	IsValid bool `json:"is_valid"`
}

func ValidateHub(hubID string) bool {
	if imsClient == nil {
		return false
	}

	req := &http.Request{
		Url: fmt.Sprintf("/validate/hub/%s", hubID),
	}
	fmt.Println("above")
	var result ValidationResponse
	_, err := imsClient.Get(req, &result)
	fmt.Println("checked")
	return err == nil && result.IsValid
}

func ValidateSKUOnHub(skuID string) bool {
	if imsClient == nil {
		return false
	}

	req := &http.Request{
		Url: fmt.Sprintf("/validate/sku/%s", skuID),
	}

	var result ValidationResponse
	_, err := imsClient.Get(req, &result)
	return err == nil && result.IsValid
}
