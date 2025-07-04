package client

import "net/http"

//go:generate mockgen -destination=mock_http_client.go -package=client github.com/gerfey/gophkeeper/internal/client HTTPClient

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

var _ HTTPClient = &http.Client{}
