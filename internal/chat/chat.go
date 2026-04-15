package chat

import (
	"fmt"
	"log/slog"
	"slices"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapter/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	state_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/state"
)

type ChatController struct {
	api             *tgbotapi.BotAPI
	stateRepo       state_repository.StateRepository
	logger          *slog.Logger
	scrapperAdapter scrapper.ScrapperAdapter
}

func NewChatController(token string, scrapperAdapter scrapper.ScrapperAdapter, stateRepo state_repository.StateRepository, logger *slog.Logger) (*ChatController, error) {
	logger.Info("init chat controller")

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Error("failed to create bot", "error", err)
		return nil, err
	}
	return &ChatController{api: api, stateRepo: stateRepo, logger: logger, scrapperAdapter: scrapperAdapter}, nil
}

func (b *ChatController) SetCommands(commands []tgbotapi.BotCommand) error {
	config := tgbotapi.SetMyCommandsConfig{Commands: commands}
	_, err := b.api.Request(config)
	return err
}

func (b *ChatController) GetUpdatesChan() tgbotapi.UpdatesChannel {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	return b.api.GetUpdatesChan(u)
}

func (b *ChatController) GetData(chatID int64) (domain.ChatData, error) {
	data, err := b.stateRepo.GetData(chatID)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (b *ChatController) SetData(chatID int64, d domain.ChatData) error {
	return b.stateRepo.SetData(chatID, d)
}

func (b *ChatController) Send(chatID int64, msg string) {
	b.api.Send(tgbotapi.NewMessage(chatID, msg))
}

func (b *ChatController) StartTrack(chatID int64) error {
	data := domain.ChatTrackData{ChatSimpleData: domain.ChatSimpleData{State: domain.LinkTrack}}
	return b.SetData(chatID, data)
}

func (b *ChatController) StopTrack(chatID int64) {
	data := domain.ChatUntrackData{ChatSimpleData: domain.ChatSimpleData{State: domain.LinkUntrack}}
	b.SetData(chatID, data)
}

func (b *ChatController) SetTrackLink(chatID int64, url string) error {
	data := domain.ChatTrackData{ChatSimpleData: domain.ChatSimpleData{State: domain.TagsTrack}, Link: domain.Link{URL: url}}
	return b.SetData(chatID, data)
}

func (b *ChatController) SetUntrackLink(chatID int64, url string) error {
	data := domain.ChatUntrackData{ChatSimpleData: domain.ChatSimpleData{State: domain.Wait}, URL: url}
	return b.SetData(chatID, data)
}

func (b *ChatController) SetTrackTags(chatID int64, tags []string) error {
	data, err := b.GetData(chatID)
	if err != nil {
		return err
	}

	trackData, ok := data.(domain.ChatTrackData)
	if !ok {
		return fmt.Errorf("data must be BotTrackData")
	}

	trackData.ChatSimpleData.State = domain.FilterTrack
	trackData.Link.Tags = tags
	return b.SetData(chatID, trackData)
}

func (b *ChatController) SetTrackFilters(chatID int64, filters []string) error {
	data, err := b.GetData(chatID)
	if err != nil {
		return err
	}

	trackData, ok := data.(domain.ChatTrackData)
	if !ok {
		return fmt.Errorf("data must be BotTrackData")
	}

	trackData.Link.Filters = filters
	return b.SetData(chatID, trackData)
}

func (b *ChatController) AddChat(chatID int64) error {
	return b.scrapperAdapter.AddChat(chatID)
}

func (b *ChatController) DeleteChat(chatID int64) error {
	return b.scrapperAdapter.DeleteChat(chatID)
}

func (b *ChatController) GetLinks(chatID int64, tag string) ([]domain.LinkWithID, error) {
	allLinks, err := b.scrapperAdapter.GetLinks(chatID)
	if err != nil {
		return nil, err
	}

	if tag == "" {
		return allLinks, nil
	}

	links := make([]domain.LinkWithID, 0)
	for _, link := range allLinks {
		if slices.Contains(link.Tags, tag) {
			links = append(links, link)
		}
	}

	return links, err
}

func (b *ChatController) AddLink(chatID int64) error {
	data, err := b.GetData(chatID)
	if err != nil {
		return err
	}

	trackData, ok := data.(domain.ChatTrackData)
	if !ok {
		return fmt.Errorf("data must be BotTrackData")
	}

	err = b.scrapperAdapter.AddLink(chatID, trackData.Link)
	if err != nil {
		return err
	}

	return b.SetData(chatID, &domain.ChatSimpleData{State: domain.Wait})
}

func (b *ChatController) DeleteLink(chatID int64) error {
	data, err := b.GetData(chatID)
	untrackData, ok := data.(domain.ChatUntrackData)
	if !ok {
		return fmt.Errorf("data must be BotUntrackData")
	}

	err = b.scrapperAdapter.DeleteLink(chatID, untrackData.URL)
	if err != nil {
		return err
	}
	return b.SetData(chatID, &domain.ChatSimpleData{State: domain.Wait})
}

func (b *ChatController) LogError(err error) {
	b.logger.Error("error ocured:", "error", err)
}

func (b *ChatController) Wait(chatID int64) error {
	data, err := b.GetData(chatID)
	if err != nil {
		return err
	}

	if data.GetState() != domain.Wait {
		b.SetData(chatID, domain.ChatSimpleData{State: domain.Wait})
		return nil
	}

	return uerrors.ErrBotAlreadyWaiting
}
