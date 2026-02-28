package storage

import (
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/TheReshkin/timer-bot/internal/config"
	"github.com/TheReshkin/timer-bot/internal/models"
)

const eventsFile = "./data/events.json"

type Storage interface {
	SaveEvent(chatID int64, event models.Event) error
	GetEvents(chatID int64) ([]models.Event, error)
	GetAllEvents() ([]models.Event, error)
	GetEvent(chatID int64, name string) (*models.Event, error)
	FindEventAcrossChats(name string, excludeChatID int64) (*models.Event, int64, error)
	EventExists(chatID int64, name string) bool
	GetUser(chatID, userID int64) (*models.User, error)
	AddEventToUser(chatID, userID int64, event models.Event) error
}

type JSONStorage struct {
	mu sync.RWMutex
}

func NewJSONStorage() *JSONStorage {
	return &JSONStorage{}
}

type ChatData struct {
	ChatID int64          `json:"chat_id"`
	Events []models.Event `json:"events"`
	Users  []models.User  `json:"users"`
}

func (s *JSONStorage) loadData() ([]ChatData, error) {
	file, err := os.Open(eventsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []ChatData{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var data []ChatData
	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *JSONStorage) saveData(data []ChatData) error {
	if _, err := os.Stat("./data"); os.IsNotExist(err) {
		err := os.Mkdir("./data", os.ModePerm)
		if err != nil {
			return err
		}
	}

	file, err := os.Create(eventsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(data)
}

func (s *JSONStorage) SaveEvent(chatID int64, event models.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.loadData()
	if err != nil {
		return err
	}

	found := false
	for i, chat := range data {
		if chat.ChatID == chatID {
			data[i].Events = append(data[i].Events, event)
			found = true
			break
		}
	}
	if !found {
		data = append(data, ChatData{
			ChatID: chatID,
			Events: []models.Event{event},
			Users:  []models.User{},
		})
	}

	return s.saveData(data)
}

func (s *JSONStorage) GetEvents(chatID int64) ([]models.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.loadData()
	if err != nil {
		return nil, err
	}

	for _, chat := range data {
		if chat.ChatID == chatID {
			return chat.Events, nil
		}
	}
	return []models.Event{}, nil
}

func (s *JSONStorage) GetAllEvents() ([]models.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.loadData()
	if err != nil {
		return nil, err
	}

	var allEvents []models.Event
	for _, chat := range data {
		allEvents = append(allEvents, chat.Events...)
	}
	return allEvents, nil
}

func (s *JSONStorage) GetEvent(chatID int64, name string) (*models.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.loadData()
	if err != nil {
		return nil, err
	}

	for _, chat := range data {
		if chat.ChatID == chatID {
			for _, event := range chat.Events {
				if event.Name == name {
					return &event, nil
				}
			}
		}
	}
	return nil, errors.New("event not found")
}

func (s *JSONStorage) FindEventAcrossChats(name string, excludeChatID int64) (*models.Event, int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.loadData()
	if err != nil {
		return nil, 0, err
	}

	// First, try to find in the test chat (prioritize it for cross-chat searches)
	testChatID := config.LoadTestChatID()
	if excludeChatID != testChatID {
		for _, chat := range data {
			if chat.ChatID == testChatID {
				for _, event := range chat.Events {
					if event.Name == name {
						return &event, testChatID, nil
					}
				}
			}
		}
	}

	// If not found in test chat, search other chats
	for _, chat := range data {
		if chat.ChatID != excludeChatID && chat.ChatID != testChatID {
			for _, event := range chat.Events {
				if event.Name == name {
					return &event, chat.ChatID, nil
				}
			}
		}
	}

	return nil, 0, errors.New("event not found")
}

func (s *JSONStorage) EventExists(chatID int64, name string) bool {
	_, err := s.GetEvent(chatID, name)
	return err == nil
}

func (s *JSONStorage) GetUser(chatID, userID int64) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.loadData()
	if err != nil {
		return nil, err
	}

	for _, chat := range data {
		if chat.ChatID == chatID {
			for _, user := range chat.Users {
				if user.UserID == userID {
					return &user, nil
				}
			}
		}
	}
	return nil, errors.New("user not found")
}

func (s *JSONStorage) AddEventToUser(chatID, userID int64, event models.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.loadData()
	if err != nil {
		return err
	}

	found := false
	for i, chat := range data {
		if chat.ChatID == chatID {
			for j, user := range chat.Users {
				if user.UserID == userID {
					data[i].Users[j].Events = append(data[i].Users[j].Events, event)
					found = true
					break
				}
			}
			if !found {
				newUser := models.User{
					UserID: userID,
					ChatID: chatID,
					Events: []models.Event{event},
				}
				data[i].Users = append(data[i].Users, newUser)
				found = true
			}
			break
		}
	}
	if !found {
		newUser := models.User{
			UserID: userID,
			ChatID: chatID,
			Events: []models.Event{event},
		}
		data = append(data, ChatData{
			ChatID: chatID,
			Events: []models.Event{},
			Users:  []models.User{newUser},
		})
	}

	return s.saveData(data)
}
