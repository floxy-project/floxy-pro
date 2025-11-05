package handlers

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rom8726/floxy-pro"
)

type HTTPHandler struct {
	name      string
	url       string
	tlsConfig *floxy.TLSConfig
	client    *http.Client
}

func NewHTTPHandler(name, url string, tlsConfig *floxy.TLSConfig) (*HTTPHandler, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}

	if tlsConfig != nil {
		if tlsConfig.SkipVerify {
			transport.TLSClientConfig.InsecureSkipVerify = true
		}

		if tlsConfig.CertFile != "" && tlsConfig.KeyFile != "" {
			cert, err := tls.LoadX509KeyPair(tlsConfig.CertFile, tlsConfig.KeyFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load client certificate: %w", err)
			}
			transport.TLSClientConfig.Certificates = []tls.Certificate{cert}
		}

		if tlsConfig.CAFile != "" {
			caCert, err := os.ReadFile(tlsConfig.CAFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate: %w", err)
			}
			caCertPool := x509.NewCertPool()
			if !caCertPool.AppendCertsFromPEM(caCert) {
				return nil, fmt.Errorf("failed to parse CA certificate")
			}
			transport.TLSClientConfig.RootCAs = caCertPool
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &HTTPHandler{
		name:      name,
		url:       url,
		tlsConfig: tlsConfig,
		client:    client,
	}, nil
}

func (h *HTTPHandler) Name() string {
	return h.name
}

func (h *HTTPHandler) Execute(
	ctx context.Context,
	stepCtx floxy.StepContext,
	input json.RawMessage,
) (json.RawMessage, error) {
	metadata := stepCtx.CloneData()

	requestBody := map[string]interface{}{
		"metadata": metadata,
		"data":     input,
	}

	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Floxy-Instance-ID", fmt.Sprintf("%d", stepCtx.InstanceID()))
	req.Header.Set("X-Floxy-Step-Name", stepCtx.StepName())
	req.Header.Set("X-Floxy-Idempotency-Key", stepCtx.IdempotencyKey())
	req.Header.Set("X-Floxy-Retry-Count", fmt.Sprintf("%d", stepCtx.RetryCount()))

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("HTTP response is empty")
	}

	var result json.RawMessage
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("HTTP response is not valid JSON: %w\nResponse: %s", err, string(body))
	}

	return result, nil
}
