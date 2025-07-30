package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config now includes the Etherscan API key.
type Config struct {
	TelegramToken   string
	SupabaseURL     string
	SupabaseKey     string
	EtherscanAPIKey string
}

// Load reads all configuration from environment variables.
func Load() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_TOKEN not set")
	}

	sbURL := os.Getenv("SUPABASE_URL")
	if sbURL == "" {
		return nil, fmt.Errorf("SUPABASE_URL not set")
	}

	sbKey := os.Getenv("SUPABASE_KEY")
	if sbKey == "" {
		return nil, fmt.Errorf("SUPABASE_KEY not set")
	}

	// Load the new key from the environment.
	etherscanKey := os.Getenv("ETHERSCAN_API_KEY")
	if etherscanKey == "" {
		// We'll log a warning but not fail, so the bot can run without the /gas command.
		log.Println("WARNING: ETHERSCAN_API_KEY not set. The /gas command will not work.")
	}

	return &Config{
		TelegramToken:   token,
		SupabaseURL:     sbURL,
		SupabaseKey:     sbKey,
		EtherscanAPIKey: etherscanKey,
	}, nil
}
