package mikrotik

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RouterConfig holds connection parameters for a single MikroTik router.
type RouterConfig struct {
	Name          string
	Host          string
	Port          string // defaults: 443 for https, 80 for http
	Scheme        string // "http" or "https"
	User          string
	Pass          string
	TLSSkipVerify bool // set true for self-signed certificates
}

// Client is a thin REST API client for RouterOS 7.x.
type Client struct {
	name       string
	baseURL    string // e.g. https://10.1.0.1:443/rest
	fileURL    string // e.g. https://10.1.0.1:443  (for file downloads)
	authHeader string
	http       *http.Client
}

// NewClient constructs a Client from a RouterConfig.
func NewClient(cfg RouterConfig) *Client {
	scheme := cfg.Scheme
	if scheme == "" {
		scheme = "http"
	}
	port := cfg.Port
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: cfg.TLSSkipVerify} //nolint:gosec

	origin := fmt.Sprintf("%s://%s:%s", scheme, cfg.Host, port)
	return &Client{
		name:       cfg.Name,
		baseURL:    origin + "/rest",
		fileURL:    origin,
		authHeader: "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.User+":"+cfg.Pass)),
		http:       &http.Client{Timeout: 30 * time.Second, Transport: transport},
	}
}

func (c *Client) Name() string { return c.name }

// Ping verifies connectivity by fetching /system/resource.
func (c *Client) Ping() error {
	_, err := c.Get("/system/resource")
	return err
}

// Get fetches a resource list or single item.
func (c *Client) Get(path string) (json.RawMessage, error) {
	return c.do(http.MethodGet, path, nil)
}

// Post executes a RouterOS command (non-CRUD action).
func (c *Client) Post(path string, body any) (json.RawMessage, error) {
	return c.do(http.MethodPost, path, body)
}

// Patch updates fields on an existing resource.
func (c *Client) Patch(path string, body any) (json.RawMessage, error) {
	return c.do(http.MethodPatch, path, body)
}

// Put creates a new resource.
func (c *Client) Put(path string, body any) (json.RawMessage, error) {
	return c.do(http.MethodPut, path, body)
}

// Delete removes a resource by its full path including ID (e.g. /ip/firewall/filter/*3).
func (c *Client) Delete(path string) error {
	_, err := c.do(http.MethodDelete, path, nil)
	return err
}

// DownloadFile downloads a file served by the router's web server (e.g. a .backup file).
func (c *Client) DownloadFile(filename string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.fileURL+"/"+filename, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.authHeader)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download %s from %s: %w", filename, c.name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: HTTP %d", filename, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (c *Client) do(method, path string, body any) (json.RawMessage, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", c.name, err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr struct {
			Error   int    `json:"error"`
			Message string `json:"message"`
			Detail  string `json:"detail"`
		}
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Message != "" {
			detail := apiErr.Detail
			if detail != "" {
				detail = " — " + detail
			}
			return nil, fmt.Errorf("RouterOS %d: %s%s", apiErr.Error, apiErr.Message, detail)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	return json.RawMessage(data), nil
}
