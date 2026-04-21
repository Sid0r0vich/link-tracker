package delivery

import (
	"fmt"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/bot"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
)

type DeliveryService struct {
	bot bot.API
}

func NewDeliveryService(bot bot.API) *DeliveryService {
	return &DeliveryService{bot: bot}
}

func (s *DeliveryService) MakeNewsletter(chatIDs []int64, url string, events []domain.Event) {
	for _, chatID := range chatIDs {
		msg := fmt.Sprintf("Получено обновление!\nСсылка: %s\n", url)

		for _, event := range events {
			data := fmt.Sprintf(
				"Тип: %s\nНазвание: %s\nОписание: %s\nПользователь: %s\nСоздано: %s\n",
				event.Type,
				event.Title,
				event.Description,
				event.Username,
				event.CreatedAt,
			)
			s.bot.Send(chatID, msg+data)
		}
	}
}
