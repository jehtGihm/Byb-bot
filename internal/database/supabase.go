package database

import (
	"context"
	"fmt"
	"log"

	"github.com/philip-857.bit/byb-bot/internal/models"
	"github.com/supabase-community/supabase-go"
)

// Client is a wrapper around the Supabase client.
type Client struct {
	*supabase.Client
}

// NewClient initializes and returns a new Supabase client wrapper.
func NewClient(supabaseURL, supabaseKey string) (*Client, error) {
	sb, err := supabase.NewClient(supabaseURL, supabaseKey, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Supabase client: %w", err)
	}

	log.Println("Successfully connected to Supabase.")
	return &Client{sb}, nil
}

// AddUser inserts a new user record into the 'members' table.
// It uses "ON CONFLICT DO NOTHING" logic via Supabase's upsert functionality.
func (c *Client) AddUser(ctx context.Context, user *models.User) error {
	// The `data` is a slice of maps or structs.
	data := []models.User{*user}

	// Using Upsert with ignoreDuplicates=true achieves "ON CONFLICT DO NOTHING".
	_, _, err := c.From("members").Upsert(data, "telegram_id", "telegram_id", "true").Execute()
	if err != nil {
		return fmt.Errorf("failed to add user to supabase: %w", err)
	}

	log.Printf("Successfully added/verified user %d in database.", user.TelegramID)
	return nil
}

// RemoveUser deletes a user record from the 'members' table based on their Telegram ID.
func (c *Client) RemoveUser(ctx context.Context, telegramID int64) error {
	// Supabase filters require string values for matching.
	idStr := fmt.Sprintf("%d", telegramID)

	_, _, err := c.From("members").Delete("", "").Eq("telegram_id", idStr).Execute()
	if err != nil {
		return fmt.Errorf("failed to remove user from supabase: %w", err)
	}

	log.Printf("Successfully removed user %d from database.", telegramID)
	return nil
}
