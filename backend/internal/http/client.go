/**
 * @Author: Nan
 * @Date: 2024/11/12 10:29
 */

package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

type Client struct {
	client *http.Client
}

func NewClient(timeout time.Duration) *Client {
	return &Client{
		client: &http.Client{Timeout: timeout},
	}
}

func (hc *Client) Get(url string, headers map[string]string) ([]byte, error) {
	return hc.doRequest("GET", url, nil, headers)
}

func (hc *Client) Post(url string, body interface{}, headers map[string]string) ([]byte, error) {
	return hc.doRequest("POST", url, body, headers)
}

func (hc *Client) Put(url string, body interface{}, headers map[string]string) ([]byte, error) {
	return hc.doRequest("PUT", url, body, headers)
}

func (hc *Client) Delete(url string, headers map[string]string) ([]byte, error) {
	return hc.doRequest("DELETE", url, nil, headers)
}

func (hc *Client) doRequest(method, url string, body interface{}, headers map[string]string) ([]byte, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	zap.L().Info("doRequest", zap.String("url", sanitizeRequestURL(url)), zap.String("method", method), zap.Int("body_bytes", len(reqBody)), zap.Int("header_count", len(headers)))

	req, err := http.NewRequest(method, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := hc.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	zap.L().Info("doRequest", zap.String("url", sanitizeRequestURL(url)), zap.String("method", method), zap.Int("status_code", resp.StatusCode), zap.Int("response_bytes", len(respBody)))
	return respBody, nil
}

func sanitizeRequestURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "<invalid-url>"
	}
	query := parsed.Query()
	for key, values := range query {
		if isSensitiveHTTPField(key) {
			for i := range values {
				values[i] = "***"
			}
			query[key] = values
		}
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func isSensitiveHTTPField(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	return strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "authorization") ||
		strings.Contains(normalized, "credential")
}
