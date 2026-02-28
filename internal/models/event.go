package models

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type EventStatus string

const (
	StatusActive   EventStatus = "active"
	StatusOutdated EventStatus = "outdated"
)

type Event struct {
	EventID     string      `json:"event_id"`
	Name        string      `json:"name"`
	Date        string      `json:"date"`
	Description string      `json:"description"`
	Status      EventStatus `json:"status"`
	ChatID      int64       `json:"chat_id"`
}

func IsValidEventName(name string) bool {
	if len(name) == 0 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, name)
	return matched
}

func IsValidDate(date string) bool {
	// Поддержка нового формата YYYY-MM-DD или YYYY-MM-DD HH:MM
	if matched, _ := regexp.MatchString(`^\d{4}-\d{1,2}-\d{1,2}( \d{1,2}:\d{2})?$`, date); matched {
		if strings.Contains(date, " ") {
			_, err := time.Parse("2006-01-02 15:04", date)
			return err == nil
		} else {
			// Для короткого формата проверяем вручную
			parts := strings.Split(date, "-")
			if len(parts) != 3 {
				return false
			}
			year, month, day := parts[0], parts[1], parts[2]

			// Добавляем ведущие нули
			if len(month) == 1 {
				month = "0" + month
			}
			if len(day) == 1 {
				day = "0" + day
			}

			formatted := year + "-" + month + "-" + day
			_, err := time.Parse("2006-01-02", formatted)
			return err == nil
		}
	}
	// Поддержка старого формата DD.MM.YYYY для обратной совместимости
	_, err := time.Parse("02.01.2006", date)
	return err == nil
}

func ParseEventDate(dateStr string) (time.Time, error) {
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.Time{}, err
	}

	// Если формат YYYY-MM-DD HH:MM
	if matched, _ := regexp.MatchString(`^\d{4}-\d{1,2}-\d{1,2} \d{1,2}:\d{2}$`, dateStr); matched {
		parsed, err := time.ParseInLocation("2006-01-02 15:04", dateStr, location)
		if err != nil {
			return time.Time{}, err
		}
		return parsed, nil
	}

	// Если формат YYYY-MM-DD (без времени), добавляем 00:00
	if matched, _ := regexp.MatchString(`^\d{4}-\d{1,2}-\d{1,2}$`, dateStr); matched {
		// Разбираем вручную для поддержки короткого формата
		parts := strings.Split(dateStr, "-")
		if len(parts) != 3 {
			return time.Time{}, fmt.Errorf("invalid date format: %s", dateStr)
		}
		year, month, day := parts[0], parts[1], parts[2]

		// Добавляем ведущие нули
		if len(month) == 1 {
			month = "0" + month
		}
		if len(day) == 1 {
			day = "0" + day
		}

		formatted := year + "-" + month + "-" + day
		parsed, err := time.Parse("2006-01-02", formatted)
		if err != nil {
			return time.Time{}, err
		}
		return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, location), nil
	}

	// Если формат DD.MM.YYYY, добавляем 00:00
	if matched, _ := regexp.MatchString(`^\d{1,2}\.\d{1,2}\.\d{4}$`, dateStr); matched {
		parsed, err := time.Parse("02.01.2006", dateStr)
		if err != nil {
			return time.Time{}, err
		}
		return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, location), nil
	}

	return time.Time{}, fmt.Errorf("unsupported date format: %s", dateStr)
}

func FormatEventDate(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

func GenerateEventID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
