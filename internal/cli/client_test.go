package cli

import "testing"

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "keeps explicit scheme", in: "https://example.com/", want: "https://example.com"},
		{name: "defaults remote host to https", in: "stanks-api.fxtun.dev", want: "https://stanks-api.fxtun.dev"},
		{name: "defaults localhost to http", in: "localhost:8080/", want: "http://localhost:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeBaseURL(tt.in); got != tt.want {
				t.Fatalf("normalizeBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
