/*
Copyright 2026 Joshua Chuah <jchuah07@gmail.com>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/joshuachuah/flightcli/internal/cache"
	"github.com/joshuachuah/flightcli/internal/provider"
	"github.com/joshuachuah/flightcli/internal/service"
)

var airportCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

// printAPIKeyError prints an actionable error message when AVIATIONSTACK_API_KEY is missing.
func printAPIKeyError() {
	fmt.Fprintln(os.Stderr, "Error: AVIATIONSTACK_API_KEY is not set.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Fix it one of two ways:")
	fmt.Fprintln(os.Stderr, "  1. Export it in your shell:")
	fmt.Fprintln(os.Stderr, "       export AVIATIONSTACK_API_KEY=your_key_here")
	fmt.Fprintln(os.Stderr, "  2. Create a .env file in the current directory:")
	fmt.Fprintln(os.Stderr, "       AVIATIONSTACK_API_KEY=your_key_here")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Get a free key at https://aviationstack.com/")
}

func requireAPIKey() (string, error) {
	apiKey := os.Getenv("AVIATIONSTACK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("AVIATIONSTACK_API_KEY is not set")
	}
	return apiKey, nil
}

func newFlightService(apiKey string, useCache bool) service.FlightService {
	var c *cache.Cache
	if useCache {
		created, err := cache.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cache disabled: %v\n", err)
		} else {
			c = created
		}
	}

	return service.FlightService{
		Provider: &provider.AviationStackProvider{APIKey: apiKey},
		Cache:    c,
	}
}

func printJSONOutput(v interface{}) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding JSON output: %w", err)
	}
	fmt.Println(string(out))
	return nil
}

func normalizeAirportCode(input, fieldName string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(input))
	if !airportCodePattern.MatchString(code) {
		return "", fmt.Errorf("invalid %s %q: use a 3-letter IATA airport code", fieldName, input)
	}
	return code, nil
}
