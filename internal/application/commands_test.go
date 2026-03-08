package application_test

import (
	"fmt"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type MockAPI struct {
	state     domain.BotState
	Responses []string
}

func (m *MockAPI) GetData(int64) (domain.BotData, error) {
	return domain.BotSimpleData{State: m.state}, nil
}
func (m *MockAPI) SetData(int64, domain.BotData) error
func (m *MockAPI) Send(chatID int64, msg string) {
	m.Responses = append(m.Responses, msg)
}
func (m *MockAPI) StartTrack(int64) error
func (m *MockAPI) StopTrack(int64)
func (m *MockAPI) SetTrackLink(int64, string) error
func (m *MockAPI) SetUntrackLink(int64, string) error
func (m *MockAPI) SetTrackTags(int64, []string) error
func (m *MockAPI) SetTrackFilters(int64, []string) error
func (m *MockAPI) AddChat(int64) error
func (m *MockAPI) DeleteChat(int64) error
func (m *MockAPI) GetLinks(int64, string) ([]domain.LinkWithID, error)
func (m *MockAPI) AddLink(int64) error
func (m *MockAPI) DeleteLink(int64) error
func (m *MockAPI) LogError(error)
func (m *MockAPI) Wait(int64) error

func TestHandleCommands(t *testing.T) {
	mockAPI := &MockAPI{}

	msgStart := tgbotapi.Message{
		Text: "/start",
		Chat: &tgbotapi.Chat{ID: 0},
		Entities: []tgbotapi.MessageEntity{
			{
				Type:   "bot_command",
				Length: len("/start"),
			},
		},
	}
	application.HandleCommand(mockAPI, &msgStart)

	if len(mockAPI.Responses) == 0 || mockAPI.Responses[0] != "Добро пожаловать! Используйте /help, чтобы посмотреть доступные команды." {
		t.Errorf("Expected greeting, got: %v", mockAPI.Responses[0])
	}

	mockAPI.Responses = nil
	msgHelp := tgbotapi.Message{
		Text: "/help",
		Chat: &tgbotapi.Chat{ID: 0},
		Entities: []tgbotapi.MessageEntity{
			{
				Type:   "bot_command",
				Length: len("/help"),
			},
		},
	}
	application.HandleCommand(mockAPI, &msgHelp)
	if len(mockAPI.Responses) == 0 || mockAPI.Responses[0] != "Список доступных команд: /help, /start" {
		t.Errorf("Expected help message, got: %v", mockAPI.Responses[0])
	}

	mockAPI.Responses = nil
	msgUnknown := tgbotapi.Message{
		Text: "/unknown",
		Chat: &tgbotapi.Chat{ID: 0},
		Entities: []tgbotapi.MessageEntity{
			{
				Type:   "bot_command",
				Length: len("/help"),
			},
		},
	}
	application.HandleCommand(mockAPI, &msgUnknown)
	fmt.Printf("mock api: %v\n", mockAPI)
	if len(mockAPI.Responses) == 0 || mockAPI.Responses[0] != "Неизвестная команда. Воспользуйтесь /help, чтобы посмотреть список доступных команд." {
		t.Errorf("Expected error message, got: %v", mockAPI.Responses[0])
	}
}
