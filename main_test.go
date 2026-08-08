package main

import "testing"

func TestResolveURL(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		href     string
		expected string
	}{
		{
			name:     "absolute https URL stays unchanged",
			base:     "https://example.com",
			href:     "https://other.com/page",
			expected: "https://other.com/page",
		},
		{
			name:     "protocol-relative URL gets https prefix",
			base:     "https://example.com",
			href:     "//cdn.example.com/script.js",
			expected: "https://cdn.example.com/script.js",
		},
		{
			name:     "root-relative path resolves against base host",
			base:     "https://example.com/some/page",
			href:     "/about",
			expected: "https://example.com/about",
		},
		{
			name:     "relative path resolves against base",
			base:     "https://example.com",
			href:     "about",
			expected: "https://example.com/about",
		},
		{
			name:     "hash-only link is skipped",
			base:     "https://example.com",
			href:     "#section",
			expected: "",
		},
		{
			name:     "mailto link is skipped",
			base:     "https://example.com",
			href:     "mailto:someone@example.com",
			expected: "",
		},
		{
			name:     "empty href is skipped",
			base:     "https://example.com",
			href:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveURL(tt.base, tt.href)
			if got != tt.expected {
				t.Errorf("resolveURL(%q, %q) = %q, want %q", tt.base, tt.href, got, tt.expected)
			}
		})
	}
}

func TestExtractLinks(t *testing.T) {
	html := `
	<html>
	<body>
		<a href="https://example.com/page1">Page 1</a>
		<a href="/page2">Page 2</a>
		<a href="#top">Top</a>
		<a href="mailto:test@example.com">Email</a>
	</body>
	</html>`

	links := extractLinks("https://example.com", html)

	expected := 2 // only page1 and page2 should survive filtering
	if len(links) != expected {
		t.Errorf("extractLinks() returned %d links, want %d. Got: %v", len(links), expected, links)
	}
}
