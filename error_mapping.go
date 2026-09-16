package main

import (
	"fmt"
	"strings"
)

// mapMCPError adds actionable context for well-known transport failures while
// preserving the original server or client error for diagnosis.
func mapMCPError(err error) string {
	if err == nil {
		return ""
	}

	message := err.Error()
	lowerMessage := strings.ToLower(message)
	switch {
	case strings.Contains(lowerMessage, "http 401"),
		strings.Contains(lowerMessage, "http 403"),
		strings.Contains(lowerMessage, "missing valid token"),
		strings.Contains(lowerMessage, "missing authorization token"):
		return fmt.Sprintf("Machbase authentication failed. Verify the configured API token and its permissions. Original error: %s", message)
	case strings.Contains(lowerMessage, "connection refused"),
		strings.Contains(lowerMessage, "no such host"):
		return fmt.Sprintf("Cannot reach the configured Machbase server. Verify the server URL and reverse-proxy path. Original error: %s", message)
	default:
		return fmt.Sprintf("Machbase request failed: %s", message)
	}
}
