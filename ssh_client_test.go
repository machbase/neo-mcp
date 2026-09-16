package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestShellQuote(t *testing.T) {
	if got := shellQuote("console.println('hello')"); got != "'console.println('\\''hello'\\'')'" {
		t.Fatalf("unexpected shell quote: %q", got)
	}
}

func TestSSHClientResolveAddressUsesServicePortAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"result":[{"Service":"shell","Address":"tcp://127.0.0.1:5652"}]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	sshClient := NewSSHClient(client, "nt_test")
	address, err := sshClient.resolveAddress(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if address != "127.0.0.1:5652" {
		t.Fatalf("unexpected address: %q", address)
	}
}

func TestSSHClientResolveAddressFailsWithoutShellService(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"result":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "nt_test", server.Client())
	sshClient := NewSSHClient(client, "nt_test")
	if _, err := sshClient.resolveAddress(context.Background()); err == nil {
		t.Fatal("expected an error when no shell service port is reported")
	}
}
