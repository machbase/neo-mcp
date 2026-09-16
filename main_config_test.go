package main

import "testing"

func TestResolveAPITokenPrefersFlag(t *testing.T) {
	if got := resolveAPIToken(" flag-token ", "env-token"); got != "flag-token" {
		t.Fatalf("unexpected token: %q", got)
	}
}

func TestResolveAPITokenUsesEnvironmentFallback(t *testing.T) {
	if got := resolveAPIToken("", " env-token "); got != "env-token" {
		t.Fatalf("unexpected token: %q", got)
	}
}

func TestParseByteSize(t *testing.T) {
	tests := map[string]int64{
		"32768": 32768,
		"32KB":  32 * 1024,
		"1MB":   1024 * 1024,
		"1GB":   1024 * 1024 * 1024,
		"2MiB":  2 * 1024 * 1024,
	}
	for input, want := range tests {
		got, err := parseByteSize(input)
		if err != nil || got != want {
			t.Fatalf("parseByteSize(%q) = %d, %v; want %d", input, got, err, want)
		}
	}
}

func TestParseByteSizeRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{"", "0", "-1", "abc", "1.5MB"} {
		if _, err := parseByteSize(input); err == nil {
			t.Fatalf("parseByteSize(%q) should fail", input)
		}
	}
}
