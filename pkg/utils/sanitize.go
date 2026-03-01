package utils

import (
	"fmt"
	"regexp"
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
