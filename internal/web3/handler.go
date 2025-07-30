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

// HandlePriceCommand fetches the price of a cryptocurrency from the CoinGecko API.
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

	apiURL := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", coinID)

	resp, err := http.Get(apiURL)
	if err != nil {
		log.Printf("Failed to call CoinGecko API: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Sorry, an error occurred while fetching the price."))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read API response body: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Sorry, an error occurred while processing the price data."))
		return
	}

	var result map[string]map[string]float64
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Failed to parse JSON from CoinGecko: %v", err)
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, "Sorry, an error occurred while parsing the price data."))
		return
	}

	if priceData, ok := result[coinID]; ok {
		price := priceData["usd"]
		priceText := fmt.Sprintf("📈 **%s Price:** `$%.2f USD`", strings.ToUpper(coinName), price)
		msg := tgbotapi.NewMessage(message.Chat.ID, priceText)
		msg.ParseMode = "Markdown"
		bot.Send(msg)
	} else {
		bot.Send(tgbotapi.NewMessage(message.Chat.ID, fmt.Sprintf("Sorry, could not find the price for '%s'.", coinName)))
	}
}

// HandleGasCommand now uses the API key from the loaded configuration.
func HandleGasCommand(bot *tgbotapi.BotAPI, db *database.Client, message *tgbotapi.Message) {
	apiKey := "D32DHB3RSUN8YU4MTUNI9Y7KRBBJI1951P"
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
