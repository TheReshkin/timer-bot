package unit

import (
	"testing"
	"time"

	"github.com/TheReshkin/timer-bot/internal/models"
)

func TestIsValidDate(t *testing.T) {
	// Тест нового формата с временем
	if !models.IsValidDate("2025-12-31 14:30") {
		t.Error("Дата с временем должна быть валидной")
	}

	// Тест нового формата без времени
	if !models.IsValidDate("2025-12-31") {
		t.Error("Дата без времени должна быть валидной")
	}

	// Тест короткого формата
	if !models.IsValidDate("2025-9-7") {
		t.Error("Короткий формат должен быть валидным")
	}

	// Тест старого формата
	if !models.IsValidDate("31.12.2025") {
		t.Error("Старый формат должен быть валидным")
	}

	// Тест невалидного формата
	if models.IsValidDate("invalid") {
		t.Error("Невалидный формат не должен проходить валидацию")
	}
}

func TestParseDateWithTime(t *testing.T) {
	// Тест формата YYYY-MM-DD HH:MM
	parsed, err := models.ParseEventDate("2025-09-07 00:00")
	if err != nil {
		t.Fatalf("Ошибка парсинга формата с временем: %v", err)
	}

	// Загружаем часовой пояс для проверки
	location := getTestLocation(t)
	expected := time.Date(2025, 9, 7, 0, 0, 0, 0, location)
	if !parsed.Equal(expected) {
		t.Errorf("Ожидалось %v, получено %v", expected, parsed)
	}
}

func TestParseDateWithoutTime(t *testing.T) {
	// Тест формата YYYY-MM-DD без времени
	parsed, err := models.ParseEventDate("2025-12-31")
	if err != nil {
		t.Fatalf("Ошибка парсинга формата без времени: %v", err)
	}

	// Загружаем часовой пояс для проверки
	location := getTestLocation(t)
	expected := time.Date(2025, 12, 31, 0, 0, 0, 0, location)
	if !parsed.Equal(expected) {
		t.Errorf("Ожидалось %v, получено %v", expected, parsed)
	}
}

func TestParseDateShortFormat(t *testing.T) {
	// Тест формата YYYY-M-D без ведущих нулей
	parsed, err := models.ParseEventDate("2025-9-7")
	if err != nil {
		t.Fatalf("Ошибка парсинга короткого формата: %v", err)
	}

	// Загружаем часовой пояс для проверки
	location := getTestLocation(t)
	expected := time.Date(2025, 9, 7, 0, 0, 0, 0, location)
	if !parsed.Equal(expected) {
		t.Errorf("Ожидалось %v, получено %v", expected, parsed)
	}
}

func TestParseDateOldFormat(t *testing.T) {
	// Тест старого формата DD.MM.YYYY (должен добавлять 00:00)
	parsed, err := models.ParseEventDate("31.12.2025")
	if err != nil {
		t.Fatalf("Ошибка парсинга старого формата: %v", err)
	}

	// Загружаем часовой пояс для проверки
	location := getTestLocation(t)
	expected := time.Date(2025, 12, 31, 0, 0, 0, 0, location)
	if !parsed.Equal(expected) {
		t.Errorf("Ожидалось %v, получено %v", expected, parsed)
	}
}

func TestTimeLeft(t *testing.T) {
	future, _ := models.ParseEventDate("2025-12-31 14:30")
	now := time.Now()
	diff := future.Sub(now)
	if diff <= 0 {
		t.Fatal("Время до события должно быть положительным")
	}
}

func TestFormatEventDate(t *testing.T) {
	testTime := time.Date(2025, 12, 31, 14, 30, 0, 0, time.UTC)
	formatted := models.FormatEventDate(testTime)
	expected := "2025-12-31 14:30"
	if formatted != expected {
		t.Errorf("Ожидалось %s, получено %s", expected, formatted)
	}
}
