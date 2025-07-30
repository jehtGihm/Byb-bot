package captcha

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/philip-857.bit/byb-bot/internal/database"
	"github.com/philip-857.bit/byb-bot/internal/models"
)

// pendingUser now only needs to track the user's presence.
var (
	pendingUsers = make(map[int64]bool)
	mu           sync.Mutex
)

const captchaTimeout = 2 * time.Minute

// HandleNewMember sends a verification message with a button.
func HandleNewMember(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	for _, user := range message.NewChatMembers {
		if user.IsBot {
			continue
		}

		// Create the verification button. The CallbackData contains the action and the target user ID.
		verifyButton := tgbotapi.NewInlineKeyboardButtonData("✅ Click here to verify", fmt.Sprintf("verify_%d", user.ID))
		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(verifyButton),
		)

		question := fmt.Sprintf("Welcome, %s! Please click the button below to prove you're human and join the community.", user.FirstName)
		msg := tgbotapi.NewMessage(message.Chat.ID, question)
		msg.ReplyMarkup = keyboard

		sentMsg, err := bot.Send(msg)
		if err != nil {
			log.Printf("Error sending verification message: %v", err)
			continue
		}

		mu.Lock()
		pendingUsers[user.ID] = true
		mu.Unlock()

		go kickUnverifiedUser(bot, db, message.Chat.ID, user.ID, sentMsg.MessageID)
	}
}

// HandleCallbackQuery processes the button click from the verification message.
func HandleCallbackQuery(bot *tgbotapi.BotAPI, db *database.Client, query *tgbotapi.CallbackQuery) {
	// The user who clicked the button
	fromUser := query.From
	// The data from the button, e.g., "verify_12345678"
	callbackData := query.Data

	parts := strings.Split(callbackData, "_")
	if len(parts) != 2 || parts[0] != "verify" {
		return // Not a verification callback
	}

	targetUserID, _ := strconv.ParseInt(parts[1], 10, 64)

	// IMPORTANT: Only the new user is allowed to click their own verification button.
	if fromUser.ID != targetUserID {
		// Send a silent, temporary message only visible to the person who clicked.
		callback := tgbotapi.NewCallback(query.ID, "This is not your verification button.")
		bot.Request(callback)
		return
	}

	mu.Lock()
	isPending, exists := pendingUsers[fromUser.ID]
	mu.Unlock()

	if exists && isPending {
		log.Printf("User %s (%d) passed button verification", fromUser.FirstName, fromUser.ID)

		// Instantly delete the verification message.
		deleteMsg := tgbotapi.NewDeleteMessage(query.Message.Chat.ID, query.Message.MessageID)
		bot.Request(deleteMsg)

		// Remove user from the pending list to prevent kicking.
		mu.Lock()
		delete(pendingUsers, fromUser.ID)
		mu.Unlock()

		// Add user to the database.
		newUser := models.User{
			TelegramID: fromUser.ID,
			FirstName:  fromUser.FirstName,
			LastName:   fromUser.LastName,
			Username:   fromUser.UserName,
		}
		if err := db.AddUser(context.Background(), &newUser); err != nil {
			log.Printf("Failed to add user to DB: %v", err)
		}

		// Send the welcome message to the group.
		sendWelcomeMessage(bot, query.Message.Chat.ID, fromUser.FirstName)

		// Answer the callback query to confirm the action.
		callback := tgbotapi.NewCallback(query.ID, "Verification successful!")
		bot.Request(callback)
	}
}

// HandleLeavingMember remains the same.
func HandleLeavingMember(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	leftUser := message.LeftChatMember
	if leftUser != nil {
		err := db.RemoveUser(context.Background(), leftUser.ID)
		if err != nil {
			log.Printf("Failed to remove user %d from DB: %v", leftUser.ID, err)
		}
	}
}

// kickUnverifiedUser kicks a user if they don't click the button in time.
func kickUnverifiedUser(bot *tgbotapi.BotAPI, db *database.Client, chatID int64, userID int64, captchaMsgID int) {
	time.Sleep(captchaTimeout)

	mu.Lock()
	defer mu.Unlock()

	if _, stillPending := pendingUsers[userID]; stillPending {
		log.Printf("Kicking user %d for failing to verify", userID)
		kickConfig := tgbotapi.KickChatMemberConfig{
			ChatMemberConfig: tgbotapi.ChatMemberConfig{ChatID: chatID, UserID: userID},
			UntilDate:        time.Now().Add(time.Minute * 5).Unix(),
		}
		bot.Request(kickConfig)

		// Also delete the original verification message on timeout.
		bot.Send(tgbotapi.NewDeleteMessage(chatID, captchaMsgID))
		delete(pendingUsers, userID)
	}
}

// sendWelcomeMessage sends the welcome message to the group.
func sendWelcomeMessage(bot *tgbotapi.BotAPI, chatID int64, firstName string) {
	welcomeText := fmt.Sprintf(`🎉 Welcome to BYB BUILDERS COMMUNITY– Block by Block! 🚀

A big welcome to our newest member, %s! They've just been verified and are now officially part of the family. 💪🏽🧱`, firstName)
	msg := tgbotapi.NewMessage(chatID, welcomeText)
	bot.Send(msg)
}
