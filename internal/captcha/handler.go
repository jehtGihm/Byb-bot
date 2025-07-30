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

var (
	pendingUsers          = make(map[int64]bool)
	lastWelcomeMessageIDs = make(map[int64]int) // Tracks the last welcome message ID per chat
	mu                    sync.Mutex
)

const captchaTimeout = 2 * time.Minute

// HandleNewMember sends a verification message with a button.
func HandleNewMember(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	for _, user := range message.NewChatMembers {
		if user.IsBot {
			continue
		}

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
	fromUser := query.From
	callbackData := query.Data

	parts := strings.Split(callbackData, "_")
	if len(parts) != 2 || parts[0] != "verify" {
		return // Not a verification callback
	}

	targetUserID, _ := strconv.ParseInt(parts[1], 10, 64)

	if fromUser.ID != targetUserID {
		callback := tgbotapi.NewCallback(query.ID, "This is not your verification button.")
		bot.Request(callback)
		return
	}

	mu.Lock()
	isPending, exists := pendingUsers[fromUser.ID]
	mu.Unlock()

	if exists && isPending {
		log.Printf("User %s (%d) passed button verification", fromUser.FirstName, fromUser.ID)

		deleteMsg := tgbotapi.NewDeleteMessage(query.Message.Chat.ID, query.Message.MessageID)
		bot.Request(deleteMsg)

		mu.Lock()
		delete(pendingUsers, fromUser.ID)
		mu.Unlock()

		newUser := models.User{
			TelegramID: fromUser.ID,
			FirstName:  fromUser.FirstName,
			LastName:   fromUser.LastName,
			Username:   fromUser.UserName,
		}
		if err := db.AddUser(context.Background(), &newUser); err != nil {
			log.Printf("Failed to add user to DB: %v", err)
		}

		sendWelcomeMessage(bot, query.Message.Chat.ID, fromUser.FirstName)

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

		bot.Send(tgbotapi.NewDeleteMessage(chatID, captchaMsgID))
		delete(pendingUsers, userID)
	}
}

// sendWelcomeMessage now deletes the previous welcome message and sends the new, detailed one.
func sendWelcomeMessage(bot *tgbotapi.BotAPI, chatID int64, firstName string) {
	mu.Lock()
	if oldMsgID, ok := lastWelcomeMessageIDs[chatID]; ok {
		bot.Request(tgbotapi.NewDeleteMessage(chatID, oldMsgID))
	}
	mu.Unlock()

	welcomeText := fmt.Sprintf(`🎉 %s Welcome to BYB BUILDERS COMMUNITY– Block by Block! 🚀

Hey there, builder! We're so glad to have you in the family. This space is where future Web3 legends are made, and you’re now officially one of us. 💪🏽🧱

Here’s what we ask from every member:

🤝 Be kind and respectful – we're a supportive family, not a battleground.

🧠 Come with the mindset to learn, grow, and build.

🚫 No insults, no F-word, no negativity – we keep it clean and empowering.

🌍 Share your journey! Feel free to introduce yourself – what do you do or want to do in Web3?


Whether you're here to explore DeFi, NFTs, DAOs, or just make new connections — you're in the right place.

Let’s build something great, block by block. 🧱🧱🧱

#BYBFam 💚`, firstName)

	msg := tgbotapi.NewMessage(chatID, welcomeText)
	sentMsg, err := bot.Send(msg)
	if err != nil {
		log.Printf("Failed to send welcome message: %v", err)
		return
	}

	mu.Lock()
	lastWelcomeMessageIDs[chatID] = sentMsg.MessageID
	mu.Unlock()
}
