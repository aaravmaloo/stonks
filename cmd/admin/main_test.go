package main

import "testing"

func TestParseStonkyArg(t *testing.T) {
	tests := []struct {
		raw  string
		want int64
	}{
		{raw: "10", want: 10_000_000},
		{raw: "10.25", want: 10_250_000},
		{raw: "-5.5", want: -5_500_000},
	}

	for _, tc := range tests {
		got, err := parseStonkyArg(tc.raw)
		if err != nil {
			t.Fatalf("parseStonkyArg(%q) error = %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("parseStonkyArg(%q) = %d, want %d", tc.raw, got, tc.want)
		}
	}
}
