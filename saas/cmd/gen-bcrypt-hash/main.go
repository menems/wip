// gen-bcrypt-hash generates a bcrypt hash for a given plaintext password.
// Used to produce password_hash values for seed migrations.
//
// Usage:
//
//	go run ./cmd/gen-bcrypt-hash <password>
//	go run ./cmd/gen-bcrypt-hash          # prompts interactively
package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	var password string

	switch len(os.Args) {
	case 1:
		fmt.Fprint(os.Stderr, "Password: ")
		if _, err := fmt.Scan(&password); err != nil {
			fmt.Fprintf(os.Stderr, "error reading password: %v\n", err)
			os.Exit(1)
		}
	case 2:
		password = os.Args[1]
	default:
		fmt.Fprintln(os.Stderr, "usage: gen-bcrypt-hash [password]")
		os.Exit(2)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(hash))
}
