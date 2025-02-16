package alert

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/spf13/viper"
)

type TeleBot struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

func newTeleBot() *TeleBot {
	// Create a new bot instance
	botToken := viper.GetString("alert.tele.token")
	chatID := viper.GetInt64("alert.tele.chat_id")
	log.Println("botToken: ", botToken, chatID)
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Panic(err)
	}

	return &TeleBot{
		bot:    bot,
		chatID: chatID,
	}
}

func (h *TeleBot) SendMessage(msg string) error {
	// Create a new message
	log.Println(h.chatID)
	text := tgbotapi.NewMessage(h.chatID, msg)

	// Send the message
	_, err := h.bot.Send(text)
	if err != nil {
		return err
	}

	return nil
}
