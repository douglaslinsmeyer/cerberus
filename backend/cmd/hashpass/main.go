package main

import (
	"fmt"
	"os"

	"github.com/cerberus/backend/internal/platform/auth"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <password>")
		os.Exit(1)
	}

	password := os.Args[1]

	hasher := auth.NewPasswordHasher()
	hash, err := hasher.Hash(password)
	if err != nil {
		fmt.Printf("Error generating hash: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Password: %s\n", password)
	fmt.Printf("Hash: %s\n", hash)
}
