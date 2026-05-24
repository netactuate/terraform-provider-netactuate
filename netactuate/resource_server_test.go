package netactuate

import (
	"fmt"
	"regexp"
	"testing"
)

func TestHostnameRegex(t *testing.T) {
	tests := []struct {
		hostname string
		valid    bool
		desc     string
	}{
		// Valid single-label hostnames
		{"a", true, "single character hostname"},
		{"z", true, "single character hostname"},
		{"0", true, "single digit hostname"},
		{"9", true, "single digit hostname"},
		{"example", true, "simple hostname"},
		{"server1", true, "hostname with number"},
		{"web-server", true, "hostname with hyphen"},
		{"my-server-123", true, "hostname with multiple hyphens and numbers"},
		{"a1b2c3", true, "hostname with mixed alphanumeric"},

		// Valid multi-label hostnames (FQDNs)
		{"example.com", true, "simple FQDN"},
		{"www.example.com", true, "subdomain FQDN"},
		{"api.v1.example.com", true, "multiple subdomain levels"},
		{"my-server.example.com", true, "subdomain with hyphen"},
		{"server1.dc2.example.com", true, "multiple labels with numbers"},
		{"a.b.c.d.e.f.g", true, "many label levels"},
		{"1.2.3.4", true, "numeric labels (though unusual for hostnames)"},
		{"web-01.prod-us-east-1.example.com", true, "complex production hostname"},

		// Invalid: starts with hyphen
		{"-server", false, "starts with hyphen"},
		{"-example.com", false, "label starts with hyphen"},
		{"server.-example.com", false, "second label starts with hyphen"},

		// Invalid: ends with hyphen
		{"server-", false, "ends with hyphen"},
		{"example-.com", false, "label ends with hyphen"},
		{"server.example-.com", false, "second label ends with hyphen"},

		// Invalid: starts or ends with dot
		{".example", false, "starts with dot"},
		{"example.", false, "ends with dot"},
		{".example.com", false, "starts with dot"},
		{"example.com.", false, "ends with dot"},

		// Invalid: double dots
		{"example..com", false, "double dot"},
		{"server..example.com", false, "double dot in middle"},

		// Invalid: special characters
		{"example_com", false, "underscore not allowed"},
		{"example@com", false, "@ symbol not allowed"},
		{"example#com", false, "# symbol not allowed"},
		{"example com", false, "space not allowed"},
		{"example/com", false, "slash not allowed"},

		// Invalid: empty string
		{"", false, "empty string"},

		// Edge cases: single character labels with dots
		{"a.b", true, "single char labels with dot"},
		{"a.b.c", true, "multiple single char labels"},

		// Edge cases: hyphen placement
		{"a-b", true, "hyphen between chars"},
		{"a-b-c", true, "multiple hyphens"},
		{"1-2-3", true, "hyphens between numbers"},
		{"a--b", true, "consecutive hyphens in middle (valid)"},

		// Realistic hostnames
		{"terraform.example.com", true, "realistic terraform hostname"},
		{"prod-web-01.us-east-1.example.com", true, "realistic production hostname"},
		{"db-master-001.internal.example.com", true, "realistic database hostname"},
		{"k8s-worker-node-42.cluster.local", true, "realistic kubernetes hostname"},
	}

	for _, tt := range tests {
		t.Run(tt.hostname, func(t *testing.T) {
			match := hostnameRegex.MatchString(tt.hostname)
			if match != tt.valid {
				t.Errorf("hostname %q: expected valid=%v, got valid=%v (%s)",
					tt.hostname, tt.valid, match, tt.desc)
			}
		})
	}
}

func TestHostnameRegexValue(t *testing.T) {
	// Verify the regex pattern is what we expect
	expectedInner := "([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\\-]*[a-zA-Z0-9])"
	expectedOuter := fmt.Sprintf("^(%s\\.)*%s$", expectedInner, expectedInner)

	actualOuter := hostnameRegex.String()

	if actualOuter != expectedOuter {
		t.Errorf("hostnameRegex pattern mismatch:\nexpected: %s\ngot:      %s",
			expectedOuter, actualOuter)
	}
}

func BenchmarkHostnameRegex(b *testing.B) {
	pattern := hostnameRegex.String()
	regex := regexp.MustCompile(pattern)
	hostname := "my-server-01.prod-us-east-1.example.com"

	b.Run("PreviousBehavior", func(b *testing.B) {

		b.ResetTimer()
		for b.Loop() {
			regexp.MatchString(pattern, hostname)
		}
	})

	b.Run("CompiledRegex", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			regex.MatchString(hostname)
		}
	})
}
