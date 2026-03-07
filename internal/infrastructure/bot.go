package infrastructure

import (
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type Bot struct {
	API   *tgbotapi.BotAPI
	state domain.BotState
}

func NewBot(token string, logger *slog.Logger) (*Bot, error) {
	logger.Info("init bot")

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Error("failed to create bot", "error", err)
		return nil, err
	}
	return &Bot{API: api}, nil
}

func (b *Bot) SetCommands(commands []tgbotapi.BotCommand) error {
	config := tgbotapi.SetMyCommandsConfig{Commands: commands}
	_, err := b.API.Request(config)
	return err
}

func (b *Bot) GetUpdatesChan() tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	return b.API.GetUpdatesChan(u)
}

func (b *Bot) GetState() domain.BotState {
	return b.state
}

func (b *Bot) Send(chatID int64, msg string) {
	b.API.Send(tgbotapi.NewMessage(chatID, msg))
}

func (b *Bot) StartTrack() {
	b.state = domain.StartTrack
}
