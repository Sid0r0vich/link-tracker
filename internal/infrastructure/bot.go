package infrastructure

import (
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository"
)

type Bot struct {
	API     *tgbotapi.BotAPI
	data    domain.BotData
	logger  *slog.Logger
	tracker repository.LinkRepository
}

func NewBot(token string, tracker repository.LinkRepository, logger *slog.Logger) (*Bot, error) {
	logger.Info("init bot")

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Error("failed to create bot", "error", err)
		return nil, err
	}
	return &Bot{API: api, data: &domain.BotSimpleData{State: domain.Wait}, logger: logger, tracker: tracker}, nil
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
	data := domain.BotTrackData{BotSimpleData: domain.BotSimpleData{State: domain.LinkTrack}}
	b.SetData(&data)
}

func (b *Bot) StopTrack() {
	data := domain.BotUntrackData{BotSimpleData: domain.BotSimpleData{State: domain.LinkUntrack}}
	b.SetData(&data)
}

func (b *Bot) SetTrackLink(link string) {
	data := domain.BotTrackData{BotSimpleData: domain.BotSimpleData{State: domain.TagsTrack}, Link: domain.Link{URL: link}}
	b.SetData(&data)
}

func (b *Bot) SetUntrackLink(url string) {
	data := domain.BotUntrackData{BotSimpleData: domain.BotSimpleData{State: domain.Wait}, URL: url}
	b.SetData(&data)
}

func (b *Bot) SetTrackTags(tags []string) error {
	data, ok := b.data.(*domain.BotTrackData)
	if !ok {
		return fmt.Errorf("data must be BotTrackData")
	}

	data.BotSimpleData.State = domain.FilterTrack
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

func (b *Bot) AddChat(chatID int64) error {
	return b.tracker.AddChat(chatID)
}

func (b *Bot) DeleteChat(chatID int64) error {
	return b.tracker.DeleteChat(chatID)
}

func (b *Bot) GetLinks(chatID int64) ([]domain.LinkWithID, error) {
	return b.tracker.GetLinks(chatID)
}

func (b *Bot) AddLink(chatID int64) error {
	data, ok := b.data.(*domain.BotTrackData)
	if !ok {
		return fmt.Errorf("data must be BotTrackData")
	}

	_, err := b.tracker.AddLink(chatID, data.Link)
	b.SetData(&domain.BotSimpleData{State: domain.Wait})

	return err
}

func (b *Bot) DeleteLink(chatID int64) error {
	data, ok := b.data.(*domain.BotUntrackData)
	if !ok {
		return fmt.Errorf("data must be BotUntrackData")
	}

	_, err := b.tracker.DeleteLink(chatID, data.URL)
	b.SetData(&domain.BotSimpleData{State: domain.Wait})

	return err
}

func (b *Bot) LogError(err error) {
	b.logger.Error("error ocured:", "error", err)
}
