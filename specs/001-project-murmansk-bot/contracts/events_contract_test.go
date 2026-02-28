package contracts

import (
	"testing"

	"github.com/TheReshkin/timer-bot/internal/models"
)

func TestSetDateContract(t *testing.T) {
	// Контракт: парсинг даты в новом формате
	dateStr := "2025-12-31 14:30"
	if !models.IsValidDate(dateStr) {
		t.Error("Дата должна быть валидной")
	}

	parsed, err := models.ParseEventDate(dateStr)
	if err != nil {
		t.Fatalf("Ошибка парсинга: %v", err)
	}

	formatted := models.FormatEventDate(parsed)
	if formatted != dateStr {
		t.Errorf("Форматирование не обратимо: ожидалось %s, получено %s", dateStr, formatted)
	}
}

func TestListEventsContract(t *testing.T) {
	// Контракт: старый формат даты должен поддерживаться
	oldDateStr := "31.12.2025"
	if !models.IsValidDate(oldDateStr) {
		t.Error("Старый формат даты должен быть валидным")
	}

	parsed, err := models.ParseEventDate(oldDateStr)
	if err != nil {
		t.Fatalf("Ошибка парсинга старого формата: %v", err)
	}

	// Должен добавить 00:00
	if parsed.Hour() != 0 || parsed.Minute() != 0 {
		t.Error("Старый формат должен добавлять 00:00")
	}
}
