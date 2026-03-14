package cmd

import (
	"os"
	"testing"
)

func TestRequireAPIKeyReturnsValueWhenPresent(t *testing.T) {
	original := os.Getenv("AVIATIONSTACK_API_KEY")
	t.Cleanup(func() {
		if original == "" {
			os.Unsetenv("AVIATIONSTACK_API_KEY")
		} else {
			os.Setenv("AVIATIONSTACK_API_KEY", original)
		}
	})

	if err := os.Setenv("AVIATIONSTACK_API_KEY", "test-key"); err != nil {
		t.Fatalf("set env: %v", err)
	}

	got, err := requireAPIKey()
	if err != nil {
		t.Fatalf("requireAPIKey returned error: %v", err)
	}
	if got != "test-key" {
		t.Fatalf("expected test-key, got %q", got)
	}
}

func TestRequireAPIKeyReturnsErrorWhenMissing(t *testing.T) {
	original := os.Getenv("AVIATIONSTACK_API_KEY")
	t.Cleanup(func() {
		if original == "" {
			os.Unsetenv("AVIATIONSTACK_API_KEY")
		} else {
			os.Setenv("AVIATIONSTACK_API_KEY", original)
		}
	})
	os.Unsetenv("AVIATIONSTACK_API_KEY")

	_, err := requireAPIKey()
	if err == nil {
		t.Fatalf("expected missing API key to return an error")
	}
}

func TestNormalizeAirportCode(t *testing.T) {
	got, err := normalizeAirportCode(" jfk ", "airport code")
	if err != nil {
		t.Fatalf("normalizeAirportCode returned error: %v", err)
	}
	if got != "JFK" {
		t.Fatalf("expected JFK, got %q", got)
	}
}

func TestNormalizeAirportCodeRejectsInvalidInput(t *testing.T) {
	_, err := normalizeAirportCode("JFK1", "airport code")
	if err == nil {
		t.Fatalf("expected invalid airport code to return an error")
	}
}
