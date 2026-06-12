package main

import (
	"os"
	"testing"
)

func TestPortEnvVar(t *testing.T) {
	t.Setenv("PORT", "9999")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if port != "9999" {
		t.Errorf("port = %q, want %q", port, "9999")
	}
}

func TestPortDefault(t *testing.T) {
	t.Setenv("PORT", "")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if port != "8080" {
		t.Errorf("default port = %q, want %q", port, "8080")
	}
}
