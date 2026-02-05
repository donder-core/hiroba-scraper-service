package main

import (
	"fmt"
	"os"

	"github.com/ap-rmit/scraper-don/internal/auth"
)

func main() {
	// Enable debug logging if DEBUG environment variable is set
	debug := os.Getenv("DEBUG") == "true"
	tokenHandler := auth.GetInstanceWithDebug(debug)

	fmt.Println("Authenticating...")
	token, err := tokenHandler.Authenticate(os.Getenv("TAIKO_USERNAME"), os.Getenv("TAIKO_PASSWORD"))
	if err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
	} else {
		fmt.Println("Authentication successful")
		fmt.Printf("Token (_token_v2): %s\n", token)
	}
}
