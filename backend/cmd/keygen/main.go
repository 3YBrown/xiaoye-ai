package main

import (
	"flag"
	"fmt"
	"google-ai-proxy/internal/auth"
	"os"
)

func main() {
	credits := flag.Int("credits", 200, "Credits to add to user account")
	flag.Parse()

	if *credits <= 0 {
		fmt.Println("Credits must be positive")
		os.Exit(1)
	}

	key, err := auth.GenerateLicenseKey(*credits)
	if err != nil {
		fmt.Printf("Error generating key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Generated License Key:")
	fmt.Println(key)
}
