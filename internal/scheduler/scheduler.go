package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	link_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/service/update"
	api "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/pkg/api/bot/rest"
)

type Scrapper interface {
	GetUpdate(string) (*domain.Update, error)
}

type Scheduler struct {
	ctx              context.Context
	linkRepo         link_repository.LinkUpdateRepository
	logger           *slog.Logger
	updater          *update.UpdateService
	scrapper         Scrapper
	sched            gocron.Scheduler
	jobDelayInterval time.Duration
}

func NewScheduler(
	linkRepo link_repository.LinkUpdateRepository,
	logger *slog.Logger,
	updater *update.UpdateService,
	scrapper Scrapper,
	jobDelayInterval time.Duration,
) (*Scheduler, error) {
	sched, err := gocron.NewScheduler()
	if err != nil {
		return nil, fmt.Errorf("creating sheduler: %w", err)
	}

	return &Scheduler{
		ctx:              context.Background(),
		linkRepo:         linkRepo,
		logger:           logger,
		updater:          updater,
		scrapper:         scrapper,
		sched:            sched,
		jobDelayInterval: jobDelayInterval,
	}, nil
}

func (s *Scheduler) LogError(err error) {
	s.logger.Error("scheduler", "error", err)
}

func (s *Scheduler) checkLinkUpdates(lurl string, linkUpd domain.LinkUpdate) error {
	s.logger.Info("checking link updates", "url", lurl)
	update, err := s.scrapper.GetUpdate(lurl)
	if err != nil {
		return fmt.Errorf("get updates from scrapper: %w", err)
	}

	oldUpdatedAt, err := s.linkRepo.GetTimeAndUpdateLink(lurl, update.UpdatedAt)
	if err != nil {
		s.logger.Error(fmt.Sprintf("get time and update link: %v", err))
		return fmt.Errorf("get link update time: %w", err)
	}

	if !oldUpdatedAt.Before(update.UpdatedAt) {
		return nil
	}

	res := api.UpdateResponse{
		Id:        update.ID,
		Url:       update.URL,
		TgChatIds: linkUpd.IDs,
	}
	if err := s.updater.SendUpdate(&res); err != nil {
		return fmt.Errorf("send update to bot: %w", err)
	}
	return nil
}

func (s *Scheduler) CheckUpdates() error {
	links, err := s.linkRepo.GetAllLinks()
	if err != nil {
		s.LogError(fmt.Errorf("get links: %w", err))
	}

	if len(links) == 0 {
		return nil
	}

	for _, linkUpd := range links {
		err = s.checkLinkUpdates(linkUpd.URL, linkUpd)
		if err != nil {
			s.LogError(fmt.Errorf("check link updates: %w", err))
		}
	}

	return nil
}

func (s *Scheduler) Start() error {
	_, err := s.sched.NewJob(
		gocron.DurationJob(s.jobDelayInterval),
		gocron.NewTask(s.CheckUpdates),
	)
	if err != nil {
		return err
	}

	s.sched.Start()
	return nil
}

func (s *Scheduler) Shutdown() error {
	return s.sched.Shutdown()
}
