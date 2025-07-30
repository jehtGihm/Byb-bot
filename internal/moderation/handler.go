package moderation

import (
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/philip-857.bit/byb-bot/internal/database"
)

// isUserAdmin checks if a given user is an administrator or creator of the chat.
func isUserAdmin(bot *tgbotapi.BotAPI, chatID int64, userID int64) bool {
	chatMember, err := bot.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: chatID,
			UserID: userID,
		},
	})
	if err != nil {
		log.Printf("Failed to get chat member for user %d in chat %d: %v", userID, chatID, err)
		return false
	}
	return chatMember.IsCreator() || chatMember.IsAdministrator()
}

// HandleWarnCommand allows an admin to warn a user by replying to their message.
func HandleWarnCommand(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	if !isUserAdmin(bot, message.Chat.ID, message.From.ID) {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "This command is for admins only."))
		return
	}
	if message.ReplyToMessage == nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Usage: Reply to a user's message with `/warn [optional reason]`."))
		return
	}

	reason := strings.TrimSpace(message.CommandArguments())
	if reason == "" {
		reason = "No reason provided."
	}
	userToWarn := message.ReplyToMessage.From
	adminName := message.From.FirstName

	warningText := fmt.Sprintf("⚠️ *Warning Issued* ⚠️\n\n*To User*: %s\n*Reason*: %s\n*By Admin*: %s", userToWarn.FirstName, reason, adminName)
	msg := tgbotapi.NewMessage(message.Chat.ID, warningText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
	log.Printf("Admin %s warned user %s for: %s", adminName, userToWarn.FirstName, reason)
}

// HandleMuteCommand allows an admin to mute a user for a specified duration.
func HandleMuteCommand(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	if !isUserAdmin(bot, message.Chat.ID, message.From.ID) {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "This command is for admins only."))
		return
	}
	if message.ReplyToMessage == nil {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Usage: Reply to a user's message with `/mute [duration]` (e.g., 1h, 2d, 30m). Default is 1 hour."))
		return
	}

	// Parse duration from arguments, default to 1 hour
	durationStr := strings.TrimSpace(message.CommandArguments())
	duration, err := time.ParseDuration(durationStr)
	if err != nil || durationStr == "" {
		duration = time.Hour * 1 // Default duration
	}

	userToMute := message.ReplyToMessage.From
	// To mute a user, we restrict their permissions until a certain time.
	// An empty ChatPermissions struct revokes all permissions.
	restrictConfig := tgbotapi.RestrictChatMemberConfig{
		ChatMemberConfig: tgbotapi.ChatMemberConfig{
			ChatID: message.Chat.ID,
			UserID: userToMute.ID,
		},
		Permissions: &tgbotapi.ChatPermissions{},
		UntilDate:   time.Now().Add(duration).Unix(),
	}

	_, err = bot.Request(restrictConfig)
	if err != nil {
		log.Printf("Failed to mute user: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "An error occurred while trying to mute the user."))
		return
	}

	muteText := fmt.Sprintf("🔇 %s has been muted for %s.", userToMute.FirstName, duration.String())
	bot.Send(tgbotapi.NewMessage(message.Chat.ID, muteText))
	log.Printf("Admin %s muted user %s for %s", message.From.FirstName, userToMute.FirstName, duration.String())
}
