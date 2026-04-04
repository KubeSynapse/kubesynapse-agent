// Package redactor removes sensitive data from logs and diagnostic output
// before sending to LLMs or external services.
//
// Implements a hybrid redaction strategy:
//   - Regex-based pattern matching for common secrets (passwords, tokens, API keys, IPs)
//   - Allowlisting of safe Kubernetes-specific fields
//
// This ensures no sensitive data leaves the cluster, which is a key differentiator
// from SaaS observability tools.
package redactor

import (
	"regexp"
	"strings"
)

// sensitivePatterns defines regex patterns for common sensitive data.
var sensitivePatterns = []*regexp.Regexp{
	// Secrets and tokens
	regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[=:]\s*\S+`),
	regexp.MustCompile(`(?i)(token|api[_-]?key|apikey|secret[_-]?key|access[_-]?key)\s*[=:]\s*\S+`),
	regexp.MustCompile(`(?i)(authorization|auth)\s*[=:]\s*(bearer\s+)?\S+`),
	regexp.MustCompile(`(?i)(aws_secret_access_key|aws_access_key_id)\s*[=:]\s*\S+`),

	// Base64-encoded secrets (common in K8s)
	regexp.MustCompile(`(?i)(data|value)\s*[=:]\s*[A-Za-z0-9+/]{20,}={0,2}`),

	// Connection strings
	regexp.MustCompile(`(?i)(mongodb|postgres|mysql|redis)://\S+`),
	regexp.MustCompile(`(?i)jdbc:\S+`),

	// IP addresses (private ranges)
	regexp.MustCompile(`\b(10\.\d{1,3}\.\d{1,3}\.\d{1,3})\b`),
	regexp.MustCompile(`\b(172\.(1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3})\b`),
	regexp.MustCompile(`\b(192\.168\.\d{1,3}\.\d{1,3})\b`),

	// Email addresses
	regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),

	// SSH keys
	regexp.MustCompile(`(?i)-----BEGIN\s+(RSA|DSA|EC|OPENSSH)\s+PRIVATE\s+KEY-----[\s\S]*?-----END`),

	// Generic hex secrets (40+ chars)
	regexp.MustCompile(`(?i)(key|secret|token)\s*[=:]\s*[0-9a-fA-F]{40,}`),
}

// redactionMarker is the replacement string for redacted content.
const redactionMarker = "[REDACTED]"

// Redactor handles sensitive data removal.
type Redactor struct {
	patterns       []*regexp.Regexp
	customPatterns []*regexp.Regexp
}

// New creates a new Redactor with default patterns.
func New() *Redactor {
	return &Redactor{
		patterns: sensitivePatterns,
	}
}

// AddPattern adds a custom regex pattern for redaction.
func (r *Redactor) AddPattern(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	r.customPatterns = append(r.customPatterns, re)
	return nil
}

// Redact removes all sensitive data from input text.
func (r *Redactor) Redact(input string) string {
	result := input

	// Apply default patterns
	for _, pattern := range r.patterns {
		result = pattern.ReplaceAllString(result, redactionMarker)
	}

	// Apply custom patterns
	for _, pattern := range r.customPatterns {
		result = pattern.ReplaceAllString(result, redactionMarker)
	}

	// Redact environment variable blocks that commonly contain secrets
	result = redactEnvVarBlock(result)

	return result
}

// RedactDiagnostics redacts all fields of a diagnostics-like map.
func (r *Redactor) RedactDiagnostics(fields map[string]string) map[string]string {
	redacted := make(map[string]string, len(fields))
	for k, v := range fields {
		redacted[k] = r.Redact(v)
	}
	return redacted
}

// redactEnvVarBlock handles multi-line env var sections in kubectl describe output.
func redactEnvVarBlock(input string) string {
	lines := strings.Split(input, "\n")
	inEnvBlock := false
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect env var blocks
		if strings.HasPrefix(trimmed, "Environment:") {
			inEnvBlock = true
			result = append(result, line)
			continue
		}

		// End of env block (next section starts, or non-indented line)
		if inEnvBlock && trimmed != "" && !strings.HasPrefix(line, "      ") && !strings.HasPrefix(line, "\t\t") {
			inEnvBlock = false
		}

		if inEnvBlock {
			// Redact environment variable values
			if idx := strings.Index(trimmed, "="); idx > 0 {
				key := trimmed[:idx]
				result = append(result, strings.Replace(line, trimmed, key+"="+redactionMarker, 1))
				continue
			}
			// Redact SecretKeyRef values
			if strings.Contains(trimmed, "SecretKeyRef") || strings.Contains(trimmed, "secret") {
				result = append(result, strings.Replace(line, trimmed, redactionMarker, 1))
				continue
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
