// Run with:
//
//	ENCRYPTION_KEY='32-character-key' go run ./scripts/decrypt-payload 'enc:...'
package main

import (
	"fmt"
	"os"

	"github.com/abhinavxd/libredesk/internal/crypto"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: ENCRYPTION_KEY=<32-byte-key> go run ./scripts/decrypt-payload <encrypted-payload>")
		os.Exit(2)
	}

	key, ok := os.LookupEnv("ENCRYPTION_KEY")
	if !ok {
		fmt.Fprintln(os.Stderr, "ENCRYPTION_KEY is required")
		os.Exit(2)
	}
	if len(key) != 32 {
		fmt.Fprintln(os.Stderr, "ENCRYPTION_KEY must be exactly 32 bytes")
		os.Exit(2)
	}

	plaintext, err := crypto.Decrypt(os.Args[1], key)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Print(plaintext)
}
