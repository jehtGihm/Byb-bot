package web3

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/philip-857.bit/byb-bot/internal/config"
	"github.com/philip-857.bit/byb-bot/internal/database"
)

// Cfg will be populated by main.go on startup.
var Cfg *config.Config

// HandlePriceCommand fetches the price and image of a cryptocurrency.
func HandlePriceCommand(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	coinName := strings.TrimSpace(message.CommandArguments())
	if coinName == "" {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Please specify a cryptocurrency. Usage: `/price bitcoin` or `/p eth`"))
		return
	}

	coinID := strings.ToLower(coinName)
	switch coinID {
	case "btc":
		coinID = "bitcoin"
	case "eth":
		coinID = "ethereum"
	}

	// Use the more detailed 'coins' endpoint to get image and market data.
	apiURL := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s", coinID)

	resp, err := http.Get(apiURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Failed to call CoinGecko API or coin not found: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Sorry, could not find data for '%s'.", coinName)))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read API response body: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Sorry, an error occurred while processing the price data."))
		return
	}

	// Define a struct to capture the detailed response from CoinGecko.
	var result struct {
		Symbol string `json:"symbol"`
		Image  struct {
			Large string `json:"large"`
		} `json:"image"`
		MarketData struct {
			CurrentPrice struct {
				USD float64 `json:"usd"`
			} `json:"current_price"`
		} `json:"market_data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Failed to parse JSON from CoinGecko: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Sorry, an error occurred while parsing the price data."))
		return
	}

	// Create the caption for the photo.
	caption := fmt.Sprintf("📈 **%s (%s) Price:**\n`$%.2f USD`",
		strings.ToUpper(coinID),
		strings.ToUpper(result.Symbol),
		result.MarketData.CurrentPrice.USD,
	)

	// Send a photo with the price as the caption.
	photoMsg := tgbotapi.NewPhoto(message.Chat.ID, tgbotapi.FileURL(result.Image.Large))
	photoMsg.Caption = caption
	photoMsg.ParseMode = "Markdown"

	if _, err := bot.Send(photoMsg); err != nil {
		log.Printf("Failed to send photo message: %v", err)
	}
}

// HandleGasCommand remains the same.
func HandleGasCommand(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	apiKey := Cfg.EtherscanAPIKey
	if apiKey == "" {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Sorry, the gas command is not configured by the administrator."))
		return
	}

	apiURL := fmt.Sprintf("https://api.etherscan.io/v2/api?chainid=1&module=gastracker&action=gasoracle&apikey=%s", apiKey)

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("Failed to call Etherscan Gas API: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Sorry, an error occurred while fetching gas fees."))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read Etherscan API response body: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Sorry, an error occurred while processing gas fee data."))
		return
	}

	var apiResponse struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  struct {
			SafeGasPrice    string `json:"SafeGasPrice"`
			ProposeGasPrice string `json:"ProposeGasPrice"`
			FastGasPrice    string `json:"FastGasPrice"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Printf("Failed to parse JSON from Etherscan: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Sorry, an error occurred while parsing gas fee data."))
		return
	}

	if apiResponse.Status != "1" {
		log.Printf("Etherscan API returned an error: %s", apiResponse.Message)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "The gas fee API returned an error. Please check your API key."))
		return
	}

	gasText := fmt.Sprintf(
		"⛽️ *Current Ethereum Gas Fees:*\n\n"+
			"🐢 *Slow (Safe):* `%s Gwei`\n"+
			"🚗 *Standard (Propose):* `%s Gwei`\n"+
			"🚀 *Fast:* `%s Gwei`",
		apiResponse.Result.SafeGasPrice,
		apiResponse.Result.ProposeGasPrice,
		apiResponse.Result.FastGasPrice,
	)

	msg := tgbotapi.NewMessage(message.Chat.ID, gasText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}
