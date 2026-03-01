package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Redactor sanitizes sensitive information from strings.
type Redactor struct {
	rules []rule
}

type rule struct {
	name    string
	re      *regexp.Regexp
	replace string
}

// DefaultRedactor is safe for concurrent use and provides standard redaction rules.
var DefaultRedactor = newDefaultRedactor()

func newDefaultRedactor() *Redactor {
	return &Redactor{
		rules: []rule{
			// --- Structured JSON fields (run first to preserve shape) ---
			{
				name:    "json_sensitive_fields",
				re:      regexp.MustCompile(`(?i)"(api_key|apikey|token|access_token|secret|client_secret|authorization|password|pwd)"(\s*:\s*)"[^"]+"`),
				replace: `"${1}"${2}"***REDACTED***"`,
			},

			// --- Authorization headers ---
			{
				name:    "bearer_token",
				re:      regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-\.]{20,}`),
				replace: "Bearer ***REDACTED***",
			},
			{
				name:    "basic_auth",
				re:      regexp.MustCompile(`(?i)basic\s+[a-zA-Z0-9+/=]{10,}`),
				replace: "Basic ***REDACTED***",
			},

			// --- Key-value style secrets ---
			{
				name:    "kv_secret",
				re:      regexp.MustCompile(`(?i)(api[_-]?key|token|access_token|auth_token|secret|client_secret|password|passwd|pwd)\s*[:=]\s*["']?[^"'\s;]+["']?`),
				replace: `${1}=***REDACTED***`,
			},

			// --- URL credentials ---
			{
				name:    "url_credentials",
				re:      regexp.MustCompile(`(?i)(https?://)[^:@\s]+:[^:@\s]+@`),
				replace: `${1}***REDACTED***@`,
			},

			// --- Private key headers ---
			{
				name:    "private_key_header",
				re:      regexp.MustCompile(`(?i)(private[_-]?key|privkey)\s*[:=]\s*["']?-----BEGIN`),
				replace: `${1}=***REDACTED***`,
			},

			// --- Connection string passwords (preserve delimiter) ---
			{
				name:    "conn_string_password",
				re:      regexp.MustCompile(`(?i)(password|pwd)(=)([^;\s&]+)(;?)`),
				replace: `${1}${2}***REDACTED***${4}`,
			},
		},
	}
}

// Redact sanitizes sensitive information from the input string.
func (r *Redactor) Redact(s string) string {
	if s == "" {
		return s
	}

	out := s
	for _, rule := range r.rules {
		out = rule.re.ReplaceAllString(out, rule.replace)
	}
	return out
}

// RedactError sanitizes an error message.
func (r *Redactor) RedactError(err error) string {
	if err == nil {
		return ""
	}
	return r.Redact(err.Error())
}

// Redactf formats and then sanitizes a string.
func (r *Redactor) Redactf(format string, args ...any) string {
	return r.Redact(fmt.Sprintf(format, args...))
}

// SanitizeError removes sensitive information from error messages using the DefaultRedactor.
// It redacts API keys, tokens, passwords, and other credentials.
func SanitizeError(err string) string {
	return DefaultRedactor.Redact(err)
}

// SanitizeErrorf is a convenience function that formats and then sanitizes an error message.
func SanitizeErrorf(format string, args ...any) string {
	return DefaultRedactor.Redactf(format, args...)
}

// RedactConfigJSON redacts sensitive fields from JSON config strings.
// This is useful when logging or returning config-related errors.
func RedactConfigJSON(jsonStr string) string {
	return DefaultRedactor.Redact(jsonStr)
}

// envVarPattern matches ${VAR} or ${VAR:-default} syntax
var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// ExpandEnvVars expands ${VAR} and ${VAR:-default} syntax in strings.
// It looks up environment variables and replaces the placeholders with their values.
// If a variable is not set and no default is provided, the placeholder is left as-is.
func ExpandEnvVars(s string) string {
	if s == "" {
		return s
	}

	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		// Extract content between ${ and }
		content := match[2 : len(match)-1]

		// Check for default value syntax: ${VAR:-default}
		if idx := strings.Index(content, ":-"); idx != -1 {
			varName := content[:idx]
			defaultValue := content[idx+2:]
			if value := os.Getenv(varName); value != "" {
				return value
			}
			return defaultValue
		}

		// Simple ${VAR} syntax
		if value := os.Getenv(content); value != "" {
			return value
		}

		// Variable not set, return original placeholder
		return match
	})
}

// ExpandEnvVarsInJSON expands environment variables in JSON byte data.
// It handles ${VAR} and ${VAR:-default} syntax throughout the JSON.
func ExpandEnvVarsInJSON(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	// Convert to string, expand, and convert back
	expanded := ExpandEnvVars(string(data))
	return []byte(expanded)
}
