package application_test

import (
	"fmt"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/application"
)

type MockAPI struct {
	Responses []string
}

func (m *MockAPI) Send(chatID int64, msg string) {
	m.Responses = append(m.Responses, msg)
}

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
