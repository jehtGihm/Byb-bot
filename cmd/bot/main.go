package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/philip-857.bit/byb-bot/internal/captcha"
	"github.com/philip-857.bit/byb-bot/internal/commands"
	"github.com/philip-857.bit/byb-bot/internal/config"
	"github.com/philip-857.bit/byb-bot/internal/database"
	"github.com/philip-857.bit/byb-bot/internal/web3"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Could not load config: %v", err)
	}

	// 2. Connect to Database
	db, err := database.NewClient(cfg.SupabaseURL, cfg.SupabaseKey)
	if err != nil {
		log.Fatalf("Could not connect to Supabase: %v", err)
	}

	// 3. Initialize Telegram Bot
	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Panic(err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Pass the loaded config to the web3 package so its handlers can use it.
	web3.Cfg = cfg

	// 4. Start Update Loop
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	log.Println("Bot is up and running. Waiting for updates...")
	for update := range updates {
		go handleUpdate(bot, db, update)
	}
}

// handleUpdate function routes all incoming messages to the correct package.
func handleUpdate(bot *tgbotapi.BotAPI, db *database.Client, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}

	switch {
	case update.Message.IsCommand():
		commands.Handle(bot, db, update.Message)
	case len(update.Message.NewChatMembers) > 0:
		captcha.HandleNewMember(bot, db, update.Message)
	case update.Message.ReplyToMessage != nil:
		captcha.HandleCaptchaReply(bot, db, update.Message)
	case update.Message.LeftChatMember != nil:
		captcha.HandleLeavingMember(bot, db, update.Message)
	}
}
