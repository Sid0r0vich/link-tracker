package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/infrastructure"
	link_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
	scrapper_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/scrapper"
)

type Scheduler struct {
	ctx      context.Context
	linkRepo link_repository.LinkUpdateRepository
	logger   *slog.Logger
	updater  *infrastructure.Updater
	scrapper scrapper_repository.Scrapper
}

func NewScheduler(
	linkRepo link_repository.LinkUpdateRepository,
	logger *slog.Logger,
	updater *infrastructure.Updater,
	scrapper scrapper_repository.Scrapper,
) *Scheduler {
	return &Scheduler{
		ctx:      context.Background(),
		linkRepo: linkRepo,
		logger:   logger,
		updater:  updater,
		scrapper: scrapper,
	}
}

func (b *Scheduler) LogError(err error) {
	b.logger.Error("scheduler:", "error", err)
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return

		case <-ticker.C:
			links, err := s.linkRepo.GetAllLinks()
			if err != nil {
				s.LogError(fmt.Errorf("get all links: %w", err))
				break
			}

			for url, linkUpd := range links {
				update, err := s.scrapper.GetUpdate(url)
				if err != nil {
					s.LogError(fmt.Errorf("get updates from scrapper: %w", err))
					continue
				}

				needToUpdate, err := s.linkRepo.GetTimeAndUpdateLink(update.URL, update.UpdatedAt)
				if err != nil {
					s.LogError(fmt.Errorf("get link update time: %w", err))
					continue
				}

				if !needToUpdate {
					continue
				}

				tgChatUpdateIDs := make([]int64, len(linkUpd.IDs))
				cnt := 0
				for k, _ := range linkUpd.IDs {
					tgChatUpdateIDs[cnt] = k
					cnt++
				}

				data := domain.UpdateResponse{
					ID:        update.ID,
					URL:       update.URL,
					Desc:      update.Desc,
					TgChatIds: tgChatUpdateIDs,
				}
				if err := s.updater.SendUpdate(&data); err != nil {
					s.LogError(fmt.Errorf("send update to bot: %w", err))
				}
			}
		}
	}
}
