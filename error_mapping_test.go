package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapMCPErrorPreservesOriginalError(t *testing.T) {
	err := errors.New("markdown.render failed: map[code:-32000 message:boom]")
	message := mapMCPError(err)
	require.Contains(t, message, "Machbase request failed")
	require.Contains(t, message, err.Error())
}

func TestMapMCPErrorAddsAuthenticationGuidance(t *testing.T) {
	err := errors.New("machbase API returned HTTP 401")
	message := mapMCPError(err)
	require.Contains(t, message, "authentication failed")
	require.Contains(t, message, err.Error())
}

func TestMapMCPErrorAddsConnectionGuidance(t *testing.T) {
	err := errors.New("Get \"http://127.0.0.1:5654/db/query\": connect: connection refused")
	message := mapMCPError(err)
	require.Contains(t, message, "Cannot reach")
	require.Contains(t, message, err.Error())
}
