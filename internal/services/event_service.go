package services

import (
	"errors"
	"time"

	"github.com/TheReshkin/timer-bot/internal/models"
	"github.com/TheReshkin/timer-bot/internal/storage"
	"go.uber.org/zap"
)

type EventService struct {
	store  storage.Storage
	logger *zap.Logger
}

func NewEventService(store storage.Storage) *EventService {
	logger, _ := zap.NewProduction()
	return &EventService{
		store:  store,
		logger: logger,
	}
}

func (s *EventService) CreateEvent(chatID int64, name, date, description string) error {
	s.logger.Info("Создание события",
		zap.Int64("chat_id", chatID),
		zap.String("event_name", name),
		zap.String("date", date))

	if !models.IsValidEventName(name) {
		s.logger.Warn("Некорректное имя события", zap.String("event_name", name))
		return errors.New("invalid event name")
	}
	if !models.IsValidDate(date) {
		s.logger.Warn("Некорректная дата", zap.String("date", date))
		return errors.New("invalid date format")
	}
	if s.store.EventExists(chatID, name) {
		s.logger.Warn("Событие уже существует",
			zap.Int64("chat_id", chatID),
			zap.String("event_name", name))
		return errors.New("duplicate event name")
	}
	event := models.Event{
		EventID:     models.GenerateEventID(),
		Name:        name,
		Date:        date,
		Description: description,
		Status:      models.StatusActive,
		ChatID:      chatID,
	}
	err := s.store.SaveEvent(chatID, event)
	if err != nil {
		s.logger.Error("Ошибка сохранения события", zap.Error(err))
		return err
	}
	s.logger.Info("Событие успешно создано",
		zap.Int64("chat_id", chatID),
		zap.String("event_name", name))
	return nil
}

func (s *EventService) ListEvents(chatID int64) ([]models.Event, error) {
	s.logger.Debug("Получение списка событий", zap.Int64("chat_id", chatID))
	events, err := s.store.GetEvents(chatID)
	if err != nil {
		s.logger.Error("Ошибка получения событий", zap.Error(err))
	}
	return events, err
}

func (s *EventService) GetAllEvents() ([]models.Event, error) {
	s.logger.Debug("Получение всех событий")
	events, err := s.store.GetAllEvents()
	if err != nil {
		s.logger.Error("Ошибка получения всех событий", zap.Error(err))
	}
	return events, err
}

func (s *EventService) GetEvent(chatID int64, name string) (*models.Event, error) {
	s.logger.Debug("Получение события",
		zap.Int64("chat_id", chatID),
		zap.String("event_name", name))
	event, err := s.store.GetEvent(chatID, name)
	if err != nil {
		s.logger.Warn("Событие не найдено",
			zap.Int64("chat_id", chatID),
			zap.String("event_name", name),
			zap.Error(err))
	}
	return event, err
}

func (s *EventService) FindEventAcrossChats(name string, excludeChatID int64) (*models.Event, int64, error) {
	s.logger.Debug("Поиск события в других чатах",
		zap.String("event_name", name),
		zap.Int64("exclude_chat_id", excludeChatID))
	event, chatID, err := s.store.FindEventAcrossChats(name, excludeChatID)
	if err != nil {
		s.logger.Warn("Событие не найдено в других чатах",
			zap.String("event_name", name),
			zap.Error(err))
	} else {
		s.logger.Info("Событие найдено в другом чате",
			zap.String("event_name", name),
			zap.Int64("found_in_chat_id", chatID))
	}
	return event, chatID, err
}

func (s *EventService) UpdateEventStatus(chatID int64, name string) error {
	s.logger.Debug("Обновление статуса события",
		zap.Int64("chat_id", chatID),
		zap.String("event_name", name))
	event, err := s.store.GetEvent(chatID, name)
	if err != nil {
		s.logger.Error("Ошибка получения события для обновления статуса", zap.Error(err))
		return err
	}
	parsedDate, err := models.ParseEventDate(event.Date)
	if err != nil {
		s.logger.Error("Ошибка парсинга даты события", zap.Error(err))
		return err
	}
	if parsedDate.Before(time.Now()) {
		event.Status = models.StatusOutdated
		err = s.store.SaveEvent(chatID, *event)
		if err != nil {
			s.logger.Error("Ошибка сохранения обновленного статуса", zap.Error(err))
			return err
		}
		s.logger.Info("Статус события обновлен на устаревший",
			zap.Int64("chat_id", chatID),
			zap.String("event_name", name))
	}
	return nil
}
