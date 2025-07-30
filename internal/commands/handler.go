package commands

import (
	"fmt"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/philip-857.bit/byb-bot/internal/database"
	"github.com/philip-857.bit/byb-bot/internal/moderation"
	"github.com/philip-857.bit/byb-bot/internal/web3"
)

// Command holds the function to be executed for a command.
type Command func(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message)

// commandRegistry holds all registered bot commands.
var commandRegistry = make(map[string]Command)

// init runs when the program starts, registering all our commands.
func init() {
	log.Println("Registering commands...")
	// User commands
	commandRegistry["start"] = handleStartCommand
	commandRegistry["rules"] = handleRulesCommand
	commandRegistry["help"] = handleHelpCommand

	// Web3 commands
	commandRegistry["price"] = web3.HandlePriceCommand
	commandRegistry["p"] = web3.HandlePriceCommand
	commandRegistry["gas"] = web3.HandleGasCommand

	// Admin commands
	commandRegistry["warn"] = moderation.HandleWarnCommand
	commandRegistry["mute"] = moderation.HandleMuteCommand
	commandRegistry["setup"] = moderation.HandleSetupCommand // New command
}

// Handle is the main router for all commands.
func Handle(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	commandName := message.Command()

	cmd, exists := commandRegistry[commandName]
	if !exists {
		return
	}

	cmd(bot, db, message)
}

// --- User Command Handler Implementations ---

func handleStartCommand(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	text := fmt.Sprintf("Hello, %s! I am the BYB Builders Bot. Use /help to see what I can do.", message.From.FirstName)
	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	bot.Send(msg)
}

func handleRulesCommand(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	rules := `*BYB BUILDERS COMMUNITY RULES* 🧱

1.  *Be Kind & Respectful*: We are a supportive family, not a battleground.
2.  *Stay On Topic*: Keep discussions related to Web3, building, and technology.
3.  *No Spam*: Unsolicited promotions or spam are strictly forbidden.
4.  *Help Each Other*: Come with a mindset to learn, grow, and build together.`

	msg := tgbotapi.NewMessage(message.Chat.ID, rules)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

func handleHelpCommand(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	helpText := `Here are the available commands:

*/start* - Welcome message
*/rules* - Show community rules
*/help* - Show this message
*/price* <coin> - Get cryptocurrency price
*/gas* - Get current Ethereum gas fees

*Admin Commands:*
*/warn* - Warn a user
*/mute* - Mute a user`
	msg := tgbotapi.NewMessage(message.Chat.ID, helpText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
