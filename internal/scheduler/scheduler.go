package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
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
	sched    gocron.Scheduler
}

func NewScheduler(
	linkRepo link_repository.LinkUpdateRepository,
	logger *slog.Logger,
	updater *infrastructure.Updater,
	scrapper scrapper_repository.Scrapper,
) (*Scheduler, error) {
	sched, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("creating sheduler: %w", err)
	}

	return &Scheduler{
		ctx:      context.Background(),
		linkRepo: linkRepo,
		logger:   logger,
		updater:  updater,
		scrapper: scrapper,
		sched:    sched,
	}, nil
}

func (b *Scheduler) LogError(err error) {
	b.logger.Error("scheduler:", "error", err)
}

func (s *Scheduler) Start() {
	s.sched.Start()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.sched.Shutdown()
			return

		case <-ticker.C:
			links, err := s.linkRepo.GetAllLinks()
			if err != nil {
				s.LogError(fmt.Errorf("get all links: %w", err))
				break
			}

			checkLinkUpdates := func(url string, linkUpd domain.LinkUpdate) {
				update, err := s.scrapper.GetUpdate(url)
				if err != nil {
					s.LogError(fmt.Errorf("get updates from scrapper: %w", err))
					return
				}

				needToUpdate, err := s.linkRepo.GetTimeAndUpdateLink(update.URL, update.UpdatedAt)
				if err != nil {
					s.LogError(fmt.Errorf("get link update time: %w", err))
					return
				}

				if !needToUpdate {
					return
				}

				tgChatUpdateIDs := make([]int64, len(linkUpd.IDs))
				cnt := 0
				for k := range linkUpd.IDs {
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

			for url, linkUpd := range links {
				_, err := s.sched.NewJob(
					gocron.DurationJob(
						10*time.Second,
					),
					gocron.NewTask(checkLinkUpdates, url, linkUpd),
				)
				if err != nil {
					s.LogError(fmt.Errorf("creating job: %w", err))
				}
			}
		}
	}
}
