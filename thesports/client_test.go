package thesports

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test_token")
	
	if client == nil {
		t.Fatal("Expected client to be created")
	}
	
	if client.apiToken != "test_token" {
		t.Errorf("Expected token to be 'test_token', got '%s'", client.apiToken)
	}
	
	if client.baseURL != DefaultBaseURL {
		t.Errorf("Expected baseURL to be '%s', got '%s'", DefaultBaseURL, client.baseURL)
	}
}

func TestNewClientWithConfig(t *testing.T) {
	config := Config{
		BaseURL:  "https://custom.api.com",
		APIToken: "custom_token",
		Timeout:  60 * time.Second,
	}
	
	client := NewClientWithConfig(config)
	
	if client == nil {
		t.Fatal("Expected client to be created")
	}
	
	if client.apiToken != "custom_token" {
		t.Errorf("Expected token to be 'custom_token', got '%s'", client.apiToken)
	}
	
	if client.baseURL != "https://custom.api.com" {
		t.Errorf("Expected baseURL to be 'https://custom.api.com', got '%s'", client.baseURL)
	}
	
	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("Expected timeout to be 60s, got %v", client.httpClient.Timeout)
	}
}

func TestAPIError(t *testing.T) {
	err := &APIError{
		Code:    404,
		Message: "Not found",
		Status:  "error",
	}
	
	expected := "API error 404: Not found"
	if err.Error() != expected {
		t.Errorf("Expected error message '%s', got '%s'", expected, err.Error())
	}
}

