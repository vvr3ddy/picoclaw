package utils

import (
	"strings"
	"testing"
)

func TestSanitizeError_APIKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "API key with equals",
			input:    "Error: api_key=sk-abc123def456ghi789",
			expected: "Error: api_key=***REDACTED***",
		},
		{
			name:     "API key with colon",
			input:    "Error: api_key:sk-abc123def456ghi789",
			expected: "Error: api_key=***REDACTED***", // Normalized to = for consistency
		},
		{
			name:     "APIKEY uppercase",
			input:    "Config has APIKEY=mysecretkey123456",
			expected: "Config has APIKEY=***REDACTED***",
		},
		{
			name:     "api-key with hyphen",
			input:    "api-key:testkey123456789",
			expected: "api-key=***REDACTED***", // Normalized to = for consistency
		},
		{
			name:     "short key also redacted for security",
			input:    "api_key=short",
			expected: "api_key=***REDACTED***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeError(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeError_Tokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "access token",
			input:    "Error: access_token=ghp_xxxxxxxxxxxxxxxxxxxx",
			expected: "Error: access_token=***REDACTED***",
		},
		{
			name:     "auth token",
			input:    "auth_token:secret123456",
			expected: "auth_token=***REDACTED***", // Normalized to = for consistency
		},
		{
			name:     "bearer token",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expected: "Authorization: Bearer ***REDACTED***",
		},
		{
			name:     "basic auth",
			input:    "Authorization: Basic dXNlcjpwYXNzd29yZA==",
			expected: "Authorization: Basic ***REDACTED***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeError(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeError_Secrets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "client secret",
			input:    "client_secret=supersecret123456789",
			expected: "client_secret=***REDACTED***",
		},
		{
			name:     "app secret",
			input:    "app_secret:myappsecret123",
			expected: "app_secret=***REDACTED***", // Normalized to = for consistency
		},
		{
			name:     "standalone secret with value",
			input:    "The secret=hiddenvalue123456",
			expected: "The secret=***REDACTED***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeError(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeError_Passwords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "password with equals",
			input:    "password=mysecretpassword",
			expected: "password=***REDACTED***",
		},
		{
			name:     "passwd variant",
			input:    "passwd:anothersecret",
			expected: "passwd=***REDACTED***", // Normalized to = for consistency
		},
		{
			name:     "pwd variant",
			input:    "pwd=short",
			expected: "pwd=***REDACTED***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeError(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeError_URLCreds(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "URL with credentials",
			input:    "Failed to connect to https://user:password@example.com/api",
			expected: "Failed to connect to https://***REDACTED***@example.com/api",
		},
		{
			name:     "connection string",
			input:    "Server=myserver;Database=mydb;password=mysecret;User Id=myuser",
			expected: "Server=myserver;Database=mydb;password=***REDACTED***;User Id=myuser", // Semicolon preserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeError(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeError_JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "JSON with api_key",
			input:    `{"api_key":"sk-abc123","name":"test"}`,
			expected: `{"api_key":"***REDACTED***","name":"test"}`,
		},
		{
			name:     "JSON with token",
			input:    `{"token":"ghp_secret123","user":"admin"}`,
			expected: `{"token":"***REDACTED***","user":"admin"}`,
		},
		{
			name:     "JSON with secret",
			input:    `{"client_secret":"supersecret","id":"123"}`,
			expected: `{"client_secret":"***REDACTED***","id":"123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeError(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeError_Empty(t *testing.T) {
	result := SanitizeError("")
	if result != "" {
		t.Errorf("SanitizeError(\"\") = %q, want empty string", result)
	}
}

func TestSanitizeError_NoSensitiveData(t *testing.T) {
	input := "This is a normal error message without any sensitive data"
	result := SanitizeError(input)
	if result != input {
		t.Errorf("SanitizeError should not modify clean strings: got %q", result)
	}
}

func TestSanitizeError_MultipleSecrets(t *testing.T) {
	// Use longer secrets that meet the minimum length requirements
	input := "Error: api_key=secret123456789012345 and token=secret987654321012345 with password=secret555666"
	result := SanitizeError(input)

	// Verify no raw secrets remain (they should all be redacted)
	if strings.Contains(result, "secret123456789012345") ||
		strings.Contains(result, "secret987654321012345") ||
		strings.Contains(result, "secret555666") {
		t.Errorf("SanitizeError did not redact all secrets: %q", result)
	}

	// Verify REDACTED markers are present
	count := strings.Count(result, "***REDACTED***")
	if count < 3 {
		t.Errorf("SanitizeError should contain at least 3 REDACTED markers, got %d: %q", count, result)
	}
}

func TestRedactConfigJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple config",
			input:    `{"api_key":"secret123","enabled":true}`,
			expected: `{"api_key":"***REDACTED***","enabled":true}`,
		},
		{
			name:     "Nested config",
			input:    `{"telegram":{"token":"bot123","enabled":true},"api_key":"mainkey"}`,
			expected: `{"telegram":{"token":"***REDACTED***","enabled":true},"api_key":"***REDACTED***"}`,
		},
		{
			name:     "Multiple sensitive fields",
			input:    `{"token":"t","secret":"s","password":"p","api_key":"a"}`,
			expected: `{"token":"***REDACTED***","secret":"***REDACTED***","password":"***REDACTED***","api_key":"***REDACTED***"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactConfigJSON(tt.input)
			if result != tt.expected {
				t.Errorf("RedactConfigJSON(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeError_PrivateKeys(t *testing.T) {
	input := "Error loading private_key: -----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA..."
	result := SanitizeError(input)

	if strings.Contains(result, "BEGIN RSA PRIVATE KEY") {
		t.Errorf("SanitizeError should redact private key header: %q", result)
	}

	if !strings.Contains(result, "***REDACTED***") {
		t.Errorf("SanitizeError should contain REDACTED marker: %q", result)
	}
}

func TestSanitizeError_AuthorizationHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Authorization header with API key - Bearer token",
			input:    "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
			expected: "Authorization: Bearer ***REDACTED***",
		},
		{
			name:     "Authorization in JSON",
			input:    `{"authorization":"bearer_token_12345abcd"}`,
			expected: `{"authorization":"***REDACTED***"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeError(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeError(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeError_LongSecretsRedacted(t *testing.T) {
	// Test that API keys with proper field names are redacted
	input := "Error occurred with api_key=sk-abcdefghijklmnopqrstuvwxyz123456"
	result := SanitizeError(input)

	// The api_key field should trigger redaction
	if strings.Contains(result, "sk-abcdefghijklmnopqrstuvwxyz123456") {
		t.Errorf("SanitizeError should redact API keys: %q", result)
	}

	if !strings.Contains(result, "***REDACTED***") {
		t.Errorf("SanitizeError should contain REDACTED marker: %q", result)
	}
}

func TestSanitizeError_ConfigLikeErrors(t *testing.T) {
	// Simulate an error that might include config dump
	input := `Failed to initialize: config={"api_key":"secret123456789012","token":"bearer_xyz123456789012","password":"hunter2secret"}`
	result := SanitizeError(input)

	// All sensitive values should be redacted
	if strings.Contains(result, "secret123456789012") ||
		strings.Contains(result, "bearer_xyz123456789012") ||
		strings.Contains(result, "hunter2secret") {
		t.Errorf("SanitizeError should redact config values: %q", result)
	}
}
