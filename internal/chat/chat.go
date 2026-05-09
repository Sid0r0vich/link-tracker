package chat

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/adapter/scrapper"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/config"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	uerrors "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/errors"
	state_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/state"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/utils"
)

//go:generate go run go.uber.org/mock/mockgen -source=chat.go -destination=mocks/mock.gen.go -package=mocks

type BotApi interface {
	Request(tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
	Send(tgbotapi.Chattable) (tgbotapi.Message, error)
	GetUpdatesChan(tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
}

type ChatController struct {
	api             BotApi
	stateRepo       state_repository.StateRepository
	logger          *slog.Logger
	scrapperAdapter scrapper.ScrapperAdapter
	clientTimeout   time.Duration
}

func NewChatController(cfg *config.Config, botApi BotApi, scrapperAdapter scrapper.ScrapperAdapter, stateRepo state_repository.StateRepository, logger *slog.Logger) (*ChatController, error) {
	logger.Info("init chat controller")

	return &ChatController{api: botApi, stateRepo: stateRepo, logger: logger, scrapperAdapter: scrapperAdapter, clientTimeout: cfg.DefaultHTTPClientTimeout}, nil
}

func (b *ChatController) SetCommands(commands []tgbotapi.BotCommand) error {
	config := tgbotapi.SetMyCommandsConfig{Commands: commands}
	_, err := b.api.Request(config)
	return err
}

func (b *ChatController) HandleUpdates(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.logger.Info("stop handle updates")
			return
		case update := <-updates:
			if update.Message == nil {
				b.logger.Error("nil message")
				continue
			}

			if update.Message.IsCommand() {
				b.logger.Info("get command", "command", update.Message.Command(), "chat_id", update.Message.Chat.ID)
				_ = bot.HandleCommand(b, update.Message)
			} else {
				b.logger.Info("get message", "message", update.Message.Text, "chat_id", update.Message.Chat.ID)
				_ = bot.HandleMessage(b, update.Message)
			}
		}
	}
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

func (b *ChatController) CheckUrl(url string) error {
	return utils.CheckUrl(url, b.clientTimeout)
}
