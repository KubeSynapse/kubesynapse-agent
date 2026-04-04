package redactor

import (
	"strings"
	"testing"
)

func TestRedact_Passwords(t *testing.T) {
	r := New()
	tests := []struct {
		name  string
		input string
		want  string // Should NOT contain the original secret
	}{
		{
			name:  "password in env var",
			input: "password=SuperSecret123",
			want:  "[REDACTED]",
		},
		{
			name:  "token in header",
			input: "Authorization: Bearer eyJhbGciOiJIUzI1NiIs",
			want:  "[REDACTED]",
		},
		{
			name:  "API key",
			input: "apikey=sk-ant-ABCDEFGHIJKLMNOP",
			want:  "[REDACTED]",
		},
		{
			name:  "AWS secret",
			input: "AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCY",
			want:  "[REDACTED]",
		},
		{
			name:  "connection string",
			input: "connecting to postgres://admin:password@db.internal:5432/mydb",
			want:  "[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.Redact(tt.input)
			if !strings.Contains(result, "[REDACTED]") {
				t.Errorf("Expected redaction marker in output, got: %s", result)
			}
		})
	}
}

func TestRedact_PreservesNonSensitive(t *testing.T) {
	r := New()
	input := "Pod payment-service-7d9f8b started successfully in namespace production"
	result := r.Redact(input)

	if result != input {
		t.Errorf("Non-sensitive data was modified.\nInput:  %s\nOutput: %s", input, result)
	}
}

func TestRedact_IPAddresses(t *testing.T) {
	r := New()
	tests := []struct {
		name  string
		input string
	}{
		{"private 10.x", "connecting to 10.0.1.50:3306"},
		{"private 172.x", "server at 172.16.0.100"},
		{"private 192.x", "host 192.168.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.Redact(tt.input)
			if !strings.Contains(result, "[REDACTED]") {
				t.Errorf("IP address should be redacted, got: %s", result)
			}
		})
	}
}

func TestRedact_EmailAddresses(t *testing.T) {
	r := New()
	input := "user john.doe@company.com logged in"
	result := r.Redact(input)

	if strings.Contains(result, "john.doe@company.com") {
		t.Errorf("Email should be redacted, got: %s", result)
	}
}

func TestRedact_CustomPattern(t *testing.T) {
	r := New()
	err := r.AddPattern(`CUSTOM-\d{6}`)
	if err != nil {
		t.Fatalf("AddPattern failed: %v", err)
	}

	input := "reference CUSTOM-123456 in log"
	result := r.Redact(input)

	if strings.Contains(result, "CUSTOM-123456") {
		t.Errorf("Custom pattern should be redacted, got: %s", result)
	}
}

func TestRedactDiagnostics(t *testing.T) {
	r := New()
	fields := map[string]string{
		"events":   "Pod crashed. token=abc123",
		"logs":     "Connected to postgres://admin:pass@db:5432",
		"describe": "Normal pod output without secrets",
	}

	result := r.RedactDiagnostics(fields)

	if strings.Contains(result["events"], "abc123") {
		t.Error("Token should be redacted in events")
	}
	if strings.Contains(result["logs"], "admin:pass") {
		t.Error("Connection string should be redacted in logs")
	}
	if result["describe"] != fields["describe"] {
		t.Error("Non-sensitive describe field should be unchanged")
	}
}
