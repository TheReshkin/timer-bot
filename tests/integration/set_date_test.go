package integration

import (
	"testing"

	"github.com/TheReshkin/timer-bot/internal/models"
)

func TestSetDateIntegration(t *testing.T) {
	// Тест валидации даты в новом формате
	if !models.IsValidDate("2025-12-31 14:30") {
		t.Error("Новый формат даты должен быть валидным")
	}

	// Тест парсинга даты
	parsed, err := models.ParseEventDate("2025-12-31 14:30")
	if err != nil {
		t.Fatalf("Ошибка парсинга даты: %v", err)
	}

	if parsed.Year() != 2025 || parsed.Month() != 12 || parsed.Day() != 31 {
		t.Error("Дата распарсена неправильно")
	}
}
