package utils

import (
	"os"
	"testing"
)

func TestExpandEnvVars(t *testing.T) {
	// Set up test environment variables
	os.Setenv("TEST_API_KEY", "secret123")
	os.Setenv("TEST_BASE_URL", "https://api.example.com")
	os.Setenv("EMPTY_VAR", "")
	defer func() {
		os.Unsetenv("TEST_API_KEY")
		os.Unsetenv("TEST_BASE_URL")
		os.Unsetenv("EMPTY_VAR")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple expansion",
			input:    "api_key: ${TEST_API_KEY}",
			expected: "api_key: secret123",
		},
		{
			name:     "multiple expansions",
			input:    "${TEST_BASE_URL}/v1 with key ${TEST_API_KEY}",
			expected: "https://api.example.com/v1 with key secret123",
		},
		{
			name:     "with default value",
			input:    "key: ${UNSET_VAR:-default_value}",
			expected: "key: default_value",
		},
		{
			name:     "default not used when set",
			input:    "key: ${TEST_API_KEY:-default}",
			expected: "key: secret123",
		},
		{
			name:     "unset variable left as-is",
			input:    "key: ${TOTALLY_UNSET_VAR}",
			expected: "key: ${TOTALLY_UNSET_VAR}",
		},
		{
			name:     "empty string is not unset",
			input:    "key: ${EMPTY_VAR:-default}",
			expected: "key: default",
		},
		{
			name:     "no placeholders",
			input:    "plain text without vars",
			expected: "plain text without vars",
		},
		{
			name:     "empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "JSON with env vars",
			input:    `{"api_key": "${TEST_API_KEY}", "url": "${TEST_BASE_URL}"}`,
			expected: `{"api_key": "secret123", "url": "https://api.example.com"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvVars(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandEnvVars(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandEnvVarsInJSON(t *testing.T) {
	os.Setenv("JSON_TEST_KEY", "mykey123")
	defer os.Unsetenv("JSON_TEST_KEY")

	input := []byte(`{"key": "${JSON_TEST_KEY}"}`)
	expected := []byte(`{"key": "mykey123"}`)

	result := ExpandEnvVarsInJSON(input)
	if string(result) != string(expected) {
		t.Errorf("ExpandEnvVarsInJSON(%q) = %q, want %q", input, result, expected)
	}
}

func TestExpandEnvVars_RealWorldConfig(t *testing.T) {
	os.Setenv("OPENROUTER_API_KEY", "sk-or-test123")
	os.Setenv("NVIDIA_API_KEY", "nvapi-test456")
	os.Setenv("CEREBRAS_API_KEY", "cerebras-test789")
	defer func() {
		os.Unsetenv("OPENROUTER_API_KEY")
		os.Unsetenv("NVIDIA_API_KEY")
		os.Unsetenv("CEREBRAS_API_KEY")
	}()

	jsonConfig := `{
		"model_list": [
			{
				"model_name": "openrouter-free",
				"model": "openrouter/free",
				"api_base": "https://openrouter.ai/api/v1",
				"api_key": "${OPENROUTER_API_KEY}"
			},
			{
				"model_name": "nemotron-4-340b",
				"model": "nvidia/llama-3.3-nemotron-super-49b-v1.5",
				"api_base": "https://integrate.api.nvidia.com/v1",
				"api_key": "${NVIDIA_API_KEY}"
			}
		]
	}`

	expanded := ExpandEnvVars(jsonConfig)

	// Verify the env vars were expanded
	if !contains(expanded, "sk-or-test123") {
		t.Error("Expected OPENROUTER_API_KEY to be expanded")
	}
	if !contains(expanded, "nvapi-test456") {
		t.Error("Expected NVIDIA_API_KEY to be expanded")
	}
	// Cerebras shouldn't be in the result
	if contains(expanded, "cerebras-test789") {
		t.Error("Cerebras key shouldn't appear in this config")
	}
	// Placeholders should be gone
	if contains(expanded, "${OPENROUTER_API_KEY}") {
		t.Error("Placeholder should have been replaced")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && string(s[0]) != "" && containsImpl(s, substr)
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
