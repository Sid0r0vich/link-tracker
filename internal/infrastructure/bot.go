package infrastructure

import (
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type Bot struct {
	API     *tgbotapi.BotAPI
	data    domain.BotData
	logger  *slog.Logger
	tracker application.Tracker
}

func NewBot(token string, logger *slog.Logger) (*Bot, error) {
	logger.Info("init bot")

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Error("failed to create bot", "error", err)
		return nil, err
	}
	return &Bot{API: api, logger: logger}, nil
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
	return b.data.GetState()
}

func (b *Bot) SetData(d domain.BotData) {
	b.data = d
}

func (b *Bot) Send(chatID int64, msg string) {
	b.API.Send(tgbotapi.NewMessage(chatID, msg))
}

func (b *Bot) StartTrack() {
	data := domain.BotTrackData{BotSimpleData: domain.BotSimpleData{State: domain.StartTrack}}
	b.SetData(&data)
}

func (b *Bot) SetTrackLink(link string) {
	data := domain.BotTrackData{BotSimpleData: domain.BotSimpleData{State: domain.LinkTrack}, Link: domain.Link{URL: link}}
	b.SetData(&data)
}

func (b *Bot) SetTrackTags(tags []string) error {
	data, ok := b.data.(*domain.BotTrackData)
	if !ok {
		return fmt.Errorf("data must be BotTrackData")
	}

	data.Link.Tags = tags
	return nil
}

func (b *Bot) SetTrackFilters(filters []string) error {
	data, ok := b.data.(*domain.BotTrackData)
	if !ok {
		return fmt.Errorf("data must be BotTrackData")
	}

	data.Link.Filters = filters
	return nil
}

func (b *Bot) AddLink(chatID int64) error {
	data, ok := b.data.(*domain.BotTrackData)
	if !ok {
		return fmt.Errorf("data must be BotTrackData")
	}

	b.tracker.AddLink(chatID, data.Link)
	b.logger.Info("data sent!", "data", data)
	return nil
}
