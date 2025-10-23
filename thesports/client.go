package thesports

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// DefaultBaseURL is the default API base URL
	DefaultBaseURL = "https://api.thesports.com"
	
	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second
)

// Client represents The Sports API client
type Client struct {
	baseURL    string
	username   string
	secret     string
	httpClient *http.Client
}

// Config holds the configuration for the API client
type Config struct {
	BaseURL  string
	Username string
	Secret   string
	Timeout  time.Duration
}

// NewClient creates a new The Sports API client
// username: your API username
// secret: your API secret key
func NewClient(username, secret string) *Client {
	return NewClientWithConfig(Config{
		BaseURL:  DefaultBaseURL,
		Username: username,
		Secret:   secret,
		Timeout:  DefaultTimeout,
	})
}

// NewClientWithConfig creates a new client with custom configuration
func NewClientWithConfig(config Config) *Client {
	if config.BaseURL == "" {
		config.BaseURL = DefaultBaseURL
	}
	if config.Timeout == 0 {
		config.Timeout = DefaultTimeout
	}

	return &Client{
		baseURL:  config.BaseURL,
		username: config.Username,
		secret:   config.Secret,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// doRequest performs an HTTP request
func (c *Client) doRequest(method, endpoint string, params url.Values) ([]byte, error) {
	// Build URL
	u, err := url.Parse(c.baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// Initialize params if nil
	if params == nil {
		params = url.Values{}
	}
	
	// Add authentication parameters (required by The Sports API)
	params.Set("user", c.username)
	params.Set("secret", c.secret)
	
	// Add query parameters
	u.RawQuery = params.Encode()

	// Create request
	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Message != "" {
			return nil, &apiErr
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// get performs a GET request
func (c *Client) get(endpoint string, params url.Values) ([]byte, error) {
	return c.doRequest(http.MethodGet, endpoint, params)
}

// APIError represents an API error response
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.Code, e.Message)
}

// APIResponse represents a generic API response
type APIResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

