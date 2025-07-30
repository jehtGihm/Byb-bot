package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/philip-857.bit/byb-bot/internal/botsetup"
	"github.com/philip-857.bit/byb-bot/internal/captcha"
	"github.com/philip-857.bit/byb-bot/internal/commands"
	"github.com/philip-857.bit/byb-bot/internal/config"
	"github.com/philip-857.bit/byb-bot/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Could not load config: %v", err)
	}

	db, err := database.NewClient(cfg.SupabaseURL, cfg.SupabaseKey)
	if err != nil {
		log.Fatalf("Could not connect to Supabase: %v", err)
	}

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Explicitly register commands after config is loaded to ensure dependencies are ready.
	commands.RegisterCommands(cfg)

	botsetup.SetDefaultCommands(bot)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	log.Println("Bot is up and running. Waiting for updates...")
	for update := range updates {
		go handleUpdate(bot, db, update)
	}
}

func handleUpdate(bot *tgbotapi.BotAPI, db *database.Client, update tgbotapi.Update) {
	// Handle button clicks (Callback Queries) first, as they are a distinct update type.
	if update.CallbackQuery != nil {
		captcha.HandleCallbackQuery(bot, db, update.CallbackQuery)
		return
	}

	// Handle all message-based updates.
	if update.Message == nil {
		return
	}

	// The switch statement now cleanly routes all message types.
	switch {
	case update.Message.IsCommand():
		commands.Handle(bot, db, update.Message)

	case len(update.Message.NewChatMembers) > 0:
		// Check if the bot itself was added to a new group.
		for _, member := range update.Message.NewChatMembers {
			if member.ID == bot.Self.ID {
				log.Printf("Bot added to new group: %s (%d)", update.Message.Chat.Title, update.Message.Chat.ID)
				botsetup.SetGroupCommands(bot, update.Message.Chat.ID)
			}
		}
		// Also handle the new members for CAPTCHA verification.
		captcha.HandleNewMember(bot, db, update.Message)

	case update.Message.LeftChatMember != nil:
		captcha.HandleLeavingMember(bot, db, update.Message)

		// REMOVED: The case for `update.Message.Chat.IsPrivate()` was removed as it's no longer needed.
		// Verification is now handled by the CallbackQuery above.
	}
}
