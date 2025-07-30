package botsetup

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SetDefaultCommands sets the general commands visible to all users in private chats.
func SetDefaultCommands(bot *tgbotapi.BotAPI) {
	userCommands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Welcome message"},
		{Command: "rules", Description: "Show community rules"},
		{Command: "help", Description: "Show list of commands"},
		{Command: "price", Description: "Get cryptocurrency price"},
		{Command: "gas", Description: "Get current Ethereum gas fees"},
	}

	config := tgbotapi.NewSetMyCommands(userCommands...)
	if _, err := bot.Request(config); err != nil {
		log.Printf("Failed to set default bot commands: %v", err)
	}
}

// SetGroupCommands sets specific commands for a group, with different lists for users and admins.
func SetGroupCommands(bot *tgbotapi.BotAPI, chatID int64) {
	// Commands for regular users in the group
	userCommands := []tgbotapi.BotCommand{
		{Command: "rules", Description: "Show community rules"},
		{Command: "help", Description: "Show list of commands"},
		{Command: "price", Description: "Get cryptocurrency price"},
		{Command: "gas", Description: "Get current Ethereum gas fees"},
	}
	userScope := tgbotapi.NewBotCommandScopeChat(chatID)
	userConfig := tgbotapi.NewSetMyCommandsWithScope(userScope, userCommands...)
	if _, err := bot.Request(userConfig); err != nil {
		log.Printf("Failed to set user commands for chat %d: %v", chatID, err)
	}

	// Commands for admins in the group (includes all user commands + admin commands)
	adminCommands := []tgbotapi.BotCommand{
		{Command: "rules", Description: "Show community rules"},
		{Command: "help", Description: "Show list of commands"},
		{Command: "price", Description: "Get cryptocurrency price"},
		{Command: "gas", Description: "Get current Ethereum gas fees"},
		{Command: "warn", Description: "(Admin) Warn a user"},
		{Command: "mute", Description: "(Admin) Mute a user"},
		{Command: "setup", Description: "(Admin) Refresh bot commands"},
	}
	adminScope := tgbotapi.NewBotCommandScopeChatAdministrators(chatID)
	adminConfig := tgbotapi.NewSetMyCommandsWithScope(adminScope, adminCommands...)
	if _, err := bot.Request(adminConfig); err != nil {
		log.Printf("Failed to set admin commands for chat %d: %v", chatID, err)
	}

	log.Printf("Successfully set commands for group %d", chatID)
}
