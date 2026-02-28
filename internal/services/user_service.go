package services

import (
	"github.com/TheReshkin/timer-bot/internal/models"
	"github.com/TheReshkin/timer-bot/internal/storage"
	"go.uber.org/zap"
)

type UserService struct {
	store  storage.Storage
	logger *zap.Logger
}

func NewUserService(store storage.Storage) *UserService {
	logger, _ := zap.NewProduction()
	return &UserService{
		store:  store,
		logger: logger,
	}
}

func (s *UserService) GetUser(chatID, userID int64) (*models.User, error) {
	s.logger.Debug("Получение пользователя",
		zap.Int64("chat_id", chatID),
		zap.Int64("user_id", userID))
	user, err := s.store.GetUser(chatID, userID)
	if err != nil {
		s.logger.Warn("Пользователь не найден",
			zap.Int64("chat_id", chatID),
			zap.Int64("user_id", userID),
			zap.Error(err))
	}
	return user, err
}

func (s *UserService) AddEventToUser(chatID, userID int64, event models.Event) error {
	s.logger.Info("Добавление события пользователю",
		zap.Int64("chat_id", chatID),
		zap.Int64("user_id", userID),
		zap.String("event_name", event.Name))
	err := s.store.AddEventToUser(chatID, userID, event)
	if err != nil {
		s.logger.Error("Ошибка добавления события пользователю", zap.Error(err))
		return err
	}
	s.logger.Info("Событие успешно добавлено пользователю",
		zap.Int64("chat_id", chatID),
		zap.Int64("user_id", userID),
		zap.String("event_name", event.Name))
	return nil
}
