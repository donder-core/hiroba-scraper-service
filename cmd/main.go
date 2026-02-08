package main

import (
	"fmt"
	"os"

	"github.com/ap-rmit/scraper-don/internal/auth"
	"github.com/joho/godotenv"
)

func init() {
	_ = godotenv.Load()
}

func main() {
	// Enable debug logging if DEBUG environment variable is set
	debug := os.Getenv("DEBUG") == "true"
	tokenHandler := auth.GetInstanceWithDebug(debug)

	taikoUsername := os.Getenv("TAIKO_USERNAME")
	taikoPassword := os.Getenv("TAIKO_PASSWORD")

	if taikoUsername == "" || taikoPassword == "" {
		fmt.Println("TAIKO_USERNAME or TAIKO_PASSWORD is not set")
		return
	}

	fmt.Println("Authenticating...")
	token, err := tokenHandler.Authenticate(taikoUsername, taikoPassword)
	if err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
	} else {
		fmt.Println("Authentication successful")
		fmt.Printf("Token (_token_v2): %s\n", token)
	}
}
