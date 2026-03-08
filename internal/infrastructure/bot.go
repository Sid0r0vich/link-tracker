package infrastructure

import (
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	link_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
	state_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/state"
)

type Bot struct {
	API       *tgbotapi.BotAPI
	stateRepo state_repository.StateRepository
	logger    *slog.Logger
	tracker   link_repository.LinkRepository
}

func NewBot(token string, tracker link_repository.LinkRepository, stateRepo state_repository.StateRepository, logger *slog.Logger) (*Bot, error) {
	logger.Info("init bot")

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Error("failed to create bot", "error", err)
		return nil, err
	}
	return &Bot{API: api, stateRepo: stateRepo, logger: logger, tracker: tracker}, nil
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

func (b *Bot) GetData(chatID int64) (domain.BotData, error) {
	data, err := b.stateRepo.GetData(chatID)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (b *Bot) SetData(chatID int64, d domain.BotData) error {
	return b.stateRepo.SetData(chatID, d)
}

func (b *Bot) Send(chatID int64, msg string) {
	b.API.Send(tgbotapi.NewMessage(chatID, msg))
}

func (b *Bot) StartTrack(chatID int64) {
	data := domain.BotTrackData{BotSimpleData: domain.BotSimpleData{State: domain.LinkTrack}}
	b.SetData(chatID, data)
}

func (b *Bot) StopTrack(chatID int64) {
	data := domain.BotUntrackData{BotSimpleData: domain.BotSimpleData{State: domain.LinkUntrack}}
	b.SetData(chatID, data)
}

func (b *Bot) SetTrackLink(chatID int64, link string) {
	data := domain.BotTrackData{BotSimpleData: domain.BotSimpleData{State: domain.TagsTrack}, Link: domain.Link{URL: link}}
	b.SetData(chatID, data)
}

func (b *Bot) SetUntrackLink(chatID int64, url string) {
	data := domain.BotUntrackData{BotSimpleData: domain.BotSimpleData{State: domain.Wait}, URL: url}
	b.SetData(chatID, data)
}

func (b *Bot) SetTrackTags(chatID int64, tags []string) error {
	data, err := b.GetData(chatID)
	if err != nil {
		return err
	}

	trackData, ok := data.(*domain.BotTrackData)
	if !ok {
		return fmt.Errorf("data must be BotTrackData")
	}

	trackData.BotSimpleData.State = domain.FilterTrack
	trackData.Link.Tags = tags
	return b.SetData(chatID, trackData)
}

func (b *Bot) SetTrackFilters(chatID int64, filters []string) error {
	data, err := b.GetData(chatID)
	if err != nil {
		return err
	}

	trackData, ok := data.(*domain.BotTrackData)
	if !ok {
		return fmt.Errorf("data must be BotTrackData")
	}

	trackData.Link.Filters = filters
	return b.SetData(chatID, trackData)
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
	data, err := b.GetData(chatID)
	if err != nil {
		return err
	}

	trackData, ok := data.(*domain.BotTrackData)
	if !ok {
		return fmt.Errorf("data must be BotTrackData")
	}

	_, err = b.tracker.AddLink(chatID, trackData.Link)
	b.SetData(chatID, &domain.BotSimpleData{State: domain.Wait})

	return err
}

func (b *Bot) DeleteLink(chatID int64) error {
	data, err := b.GetData(chatID)
	untrackData, ok := data.(*domain.BotUntrackData)
	if !ok {
		return fmt.Errorf("data must be BotUntrackData")
	}

	_, err = b.tracker.DeleteLink(chatID, untrackData.URL)
	b.SetData(chatID, &domain.BotSimpleData{State: domain.Wait})

	return err
}

func (b *Bot) LogError(err error) {
	b.logger.Error("error ocured:", "error", err)
}

func (b *Bot) Wait(chatID int64) error {
	data, err := b.GetData(chatID)
	if err != nil {
		return err
	}

	if data.GetState() != domain.Wait {
		b.SetData(chatID, domain.BotSimpleData{State: domain.Wait})
		return nil
	}

	return application.ErrBotAlreadyWaiting
}
