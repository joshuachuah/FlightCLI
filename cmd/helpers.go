/*
Copyright © 2026 Joshua Chuah <jchuah07@gmail.com>
*/
package cmd

import (
	"fmt"
	"os"
)

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
