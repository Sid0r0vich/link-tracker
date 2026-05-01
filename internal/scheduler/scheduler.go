package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/domain"
	link_repository "gitlab.education.tbank.ru/backend-academy-go-2025/homeworks/link-tracker/internal/repository/link"
)

type Scrapper interface {
	GetUpdate(string) (*domain.Update, error)
}

type Updater interface {
	SendUpdate(data *domain.UpdateMessage) error
}

type Scheduler struct {
	ctx              context.Context
	linkRepo         link_repository.LinkUpdateRepository
	logger           *slog.Logger
	updater          Updater
	scrapper         Scrapper
	sched            gocron.Scheduler
	jobDelayInterval time.Duration
}

func NewScheduler(
	linkRepo link_repository.LinkUpdateRepository,
	logger *slog.Logger,
	updater Updater,
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
	s.logger.Info("get link update time", "url", lurl, "old_updated_at", oldUpdatedAt, "new_updated_at", update.UpdatedAt)

	if !oldUpdatedAt.Before(update.UpdatedAt) {
		return nil
	}

	s.logger.Info("get updates", "url", lurl, "pulls", update.Data)
	data := make([]domain.Event, 0)
	for _, event := range update.Data {
		if event.CreatedAt.After(oldUpdatedAt) && !event.CreatedAt.After(update.UpdatedAt) {
			data = append(data, event)
		}
	}

	res := domain.UpdateMessage{
		Id:        update.ID,
		Url:       update.URL,
		TgChatIds: linkUpd.IDs,
		Data:      data,
	}
	if err := s.updater.SendUpdate(&res); err != nil {
		return fmt.Errorf("send update to bot: %w", err)
	}
	return nil
}

func (s *Scheduler) CheckUpdates() error {
	lastID := int64(0)

	for {
		links, newLastID, err := s.linkRepo.GetLinkBatch(lastID)
		if err != nil {
			s.LogError(fmt.Errorf("get link batch: %w", err))
		}

		if len(links) == 0 {
			break
		}

		lastID = newLastID

		for _, linkUpd := range links {
			err = s.checkLinkUpdates(linkUpd.URL, linkUpd)
			if err != nil {
				s.LogError(fmt.Errorf("check link updates: %w", err))
			}
		}
	}

	return nil
}

func (s *Scheduler) Start() error {
	_, err := s.sched.NewJob(
		gocron.DurationJob(s.jobDelayInterval),
		gocron.NewTask(s.CheckUpdates),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
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
