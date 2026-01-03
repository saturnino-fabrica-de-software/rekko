package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// calc_api_hash.go - Utility to calculate SHA256 hash for API keys
//
// Usage:
//   go run scripts/calc_api_hash.go <api_key>
//
// Example:
//   go run scripts/calc_api_hash.go rekko_test_devdevdevdevdevdevdevdevdevdev00
//
// Output:
//   adf716ab3ebb2a1138973de4a44fe454c05c0d070e897fc55220af74807b25ae

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run calc_api_hash.go <api_key>")
		fmt.Println("")
		fmt.Println("Example:")
		fmt.Println("  go run scripts/calc_api_hash.go rekko_test_devdevdevdevdevdevdevdevdevdev00")
		os.Exit(1)
	}

	apiKey := os.Args[1]
	hash := sha256.Sum256([]byte(apiKey))
	hashHex := hex.EncodeToString(hash[:])

	fmt.Printf("API Key: %s\n", apiKey)
	fmt.Printf("SHA256:  %s\n", hashHex)
}
