package captcha

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	// Corrected import paths to match your go.mod and project structure
	"github.com/philip-857.bit/byb-bot/internal/database"
	"github.com/philip-857.bit/byb-bot/internal/models"
)

type pendingUser struct {
	Answer    int
	MessageID int
}

var (
	pendingUsers = make(map[int64]pendingUser)
	mu           sync.Mutex
)

const captchaTimeout = 2 * time.Minute

// HandleNewMember sends a CAPTCHA to a new user.
// Corrected to use the Supabase client wrapper: *database.Client
func HandleNewMember(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	for _, user := range message.NewChatMembers {
		if user.IsBot {
			continue
		}

		a := rand.Intn(10) + 1
		b := rand.Intn(10) + 1
		answer := a + b

		question := fmt.Sprintf("🤖 Welcome, %s! To verify you're human, please answer: %d + %d?", user.FirstName, a, b)
		msg := tgbotapi.NewMessage(message.Chat.ID, question)
		msg.ReplyToMessageID = message.MessageID

		sentMsg, err := bot.Send(msg)
		if err != nil {
			log.Printf("Error sending CAPTCHA: %v", err)
			continue
		}

		mu.Lock()
		pendingUsers[user.ID] = pendingUser{Answer: answer, MessageID: sentMsg.MessageID}
		mu.Unlock()

		go kickUnverifiedUser(bot, db, message.Chat.ID, user.ID, sentMsg.MessageID)
	}
}

// HandleCaptchaReply checks the user's answer.
// Corrected to use the Supabase client wrapper: *database.Client
func HandleCaptchaReply(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	userID := message.From.ID

	mu.Lock()
	pending, exists := pendingUsers[userID]
	mu.Unlock()

	if !exists || message.ReplyToMessage == nil || message.ReplyToMessage.MessageID != pending.MessageID {
		return // Not a valid CAPTCHA reply
	}

	userInput, err := strconv.Atoi(message.Text)
	if err != nil {
		return // Not a number
	}

	if userInput == pending.Answer {
		// On success, clean up, add to DB, and welcome
		log.Printf("User %s passed CAPTCHA", message.From.FirstName)
		mu.Lock()
		delete(pendingUsers, userID)
		mu.Unlock()

		// Delete CAPTCHA question and user's answer
		bot.Send(tgbotapi.NewDeleteMessage(message.Chat.ID, pending.MessageID))
		bot.Send(tgbotapi.NewDeleteMessage(message.Chat.ID, message.MessageID))

		// Add user to database
		newUser := models.User{
			TelegramID: message.From.ID,
			FirstName:  message.From.FirstName,
			LastName:   message.From.LastName,
			Username:   message.From.UserName,
		}
		if err := db.AddUser(context.Background(), &newUser); err != nil {
			log.Printf("Failed to add user to DB: %v", err)
		}

		// Send the new, detailed welcome message
		sendWelcomeMessage(bot, message.Chat.ID, message.From.FirstName)
	}
}

// HandleLeavingMember handles when a user leaves or is kicked.
// Corrected to use the Supabase client wrapper: *database.Client
func HandleLeavingMember(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	leftUser := message.LeftChatMember
	if leftUser != nil {
		err := db.RemoveUser(context.Background(), leftUser.ID)
		if err != nil {
			log.Printf("Failed to remove user %d from DB: %v", leftUser.ID, err)
		}
	}
}

// kickUnverifiedUser kicks a user if they fail to solve the CAPTCHA in time.
// Corrected to use the Supabase client wrapper: *database.Client
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

// sendWelcomeMessage sends the detailed community welcome message.
func sendWelcomeMessage(bot *tgbotapi.BotAPI, chatID int64, firstName string) {
	// Updated welcome message text
	welcomeText := fmt.Sprintf(`🎉 Welcome to BYB BUILDERS COMMUNITY– Block by Block! 🚀

Hey there, %s! We're so glad to have you in the family. This space is where future Web3 legends are made, and you’re now officially one of us. 💪🏽🧱

Here’s what we ask from every member:

🤝 Be kind and respectful – we're a supportive family, not a battleground.
🧠 Come with the mindset to learn, grow, and build.
🚫 No insults, no F-word, no negativity – we keep it clean and empowering.
🌍 Share your journey! Feel free to introduce yourself – what do you do or want to do in Web3?

Whether you're here to explore DeFi, NFTs, DAOs, or just make new connections — you're in the right place.

Let’s build something great, block by block. 🧱🧱🧱

#BYBFam 💚`, firstName)

	msg := tgbotapi.NewMessage(chatID, welcomeText)
	bot.Send(msg)
}
