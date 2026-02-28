package integration

import (
	"testing"

	"github.com/TheReshkin/timer-bot/internal/models"
)

func TestListIntegration(t *testing.T) {
	// Тест форматирования даты
	parsed, err := models.ParseEventDate("2025-12-31 14:30")
	if err != nil {
		t.Fatalf("Ошибка парсинга: %v", err)
	}

	testTime := models.FormatEventDate(parsed)
	expected := "2025-12-31 14:30"
	if testTime != expected {
		t.Errorf("Ожидалось %s, получено %s", expected, testTime)
	}
}
