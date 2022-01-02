package networking

import (
	"bytes"
	"context"
	"net/http"
)

// SendSoap send soap message
func SendSoap(httpClient *http.Client, endpoint, message string) (*http.Response, error) {
	resp, err := httpClient.Post(endpoint, "application/soap+xml; charset=utf-8", bytes.NewBufferString(message))
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// SendSoapWithCtx send soap message with context
func SendSoapWithCtx(ctx context.Context, httpClient *http.Client, endpoint, message string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBufferString(message))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/soap+xml; charset=utf-8")
	resp, err := httpClient.Do(req)
	if err != nil {
		return resp, err
	}

	return resp, nil
}
