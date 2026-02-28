package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// ──────────────────────────── pending events (ожидание выбора даты/времени) ────────────────────────────

// pendingEvent хранит данные о событии, для которого ещё не выбрана дата/время.
type pendingEvent struct {
	Name        string
	Description string
	ChatID      int64
	UserID      int64
	Date        string // "YYYY-MM-DD" — заполняется после выбора дня
	Hour        int    // 0-23, -1 пока не выбран
}

var (
	pendingEvents   = make(map[string]*pendingEvent)
	pendingEventsMu sync.Mutex

	// awaitingName: пользователи, ожидающие ввода названия события после /set_date
	awaitingName   = make(map[string]bool)
	awaitingNameMu sync.Mutex
)

func pendingKey(chatID, userID int64) string {
	return fmt.Sprintf("%d:%d", chatID, userID)
}

func setPending(chatID, userID int64, pe *pendingEvent) {
	pendingEventsMu.Lock()
	defer pendingEventsMu.Unlock()
	pendingEvents[pendingKey(chatID, userID)] = pe
}

func getPending(chatID, userID int64) *pendingEvent {
	pendingEventsMu.Lock()
	defer pendingEventsMu.Unlock()
	return pendingEvents[pendingKey(chatID, userID)]
}

func deletePending(chatID, userID int64) {
	pendingEventsMu.Lock()
	defer pendingEventsMu.Unlock()
	delete(pendingEvents, pendingKey(chatID, userID))
}

// ──────────────────────────── ожидание ввода названия события ────────────────────────────

func setAwaitingName(chatID, userID int64) {
	awaitingNameMu.Lock()
	defer awaitingNameMu.Unlock()
	awaitingName[pendingKey(chatID, userID)] = true
}

func isAwaitingName(chatID, userID int64) bool {
	awaitingNameMu.Lock()
	defer awaitingNameMu.Unlock()
	return awaitingName[pendingKey(chatID, userID)]
}

func clearAwaitingName(chatID, userID int64) {
	awaitingNameMu.Lock()
	defer awaitingNameMu.Unlock()
	delete(awaitingName, pendingKey(chatID, userID))
}

// ──────────────────────────── генерация inline-календаря ────────────────────────────

var russianMonths = [12]string{
	"Январь", "Февраль", "Март", "Апрель",
	"Май", "Июнь", "Июль", "Август",
	"Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
}

var shortWeekdays = [7]string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}

// today возвращает сегодняшнюю дату без времени.
func today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// buildCalendar создаёт inline-клавиатуру с календарём. Прошедшие даты неактивны.
func buildCalendar(year int, month time.Month) *tgmodels.InlineKeyboardMarkup {
	rows := [][]tgmodels.InlineKeyboardButton{}
	todayDate := today()

	// Не даём листать назад дальше текущего месяца
	canPrev := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).After(
		time.Date(todayDate.Year(), todayDate.Month(), 1, 0, 0, 0, 0, time.UTC))

	prevBtn := tgmodels.InlineKeyboardButton{Text: " ", CallbackData: "cal:ignore"}
	if canPrev {
		prevBtn = tgmodels.InlineKeyboardButton{Text: "◀", CallbackData: fmt.Sprintf("cal:prev:%d:%d", year, int(month))}
	}

	header := []tgmodels.InlineKeyboardButton{
		prevBtn,
		{Text: fmt.Sprintf("%s %d", russianMonths[month-1], year), CallbackData: "cal:ignore"},
		{Text: "▶", CallbackData: fmt.Sprintf("cal:next:%d:%d", year, int(month))},
	}
	rows = append(rows, header)

	// Дни недели
	weekRow := make([]tgmodels.InlineKeyboardButton, 7)
	for i, d := range shortWeekdays {
		weekRow[i] = tgmodels.InlineKeyboardButton{Text: d, CallbackData: "cal:ignore"}
	}
	rows = append(rows, weekRow)

	// Сетка дней
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	startOffset := int(firstDay.Weekday()+6) % 7
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	day := 1
	for week := 0; day <= daysInMonth; week++ {
		row := make([]tgmodels.InlineKeyboardButton, 7)
		for i := 0; i < 7; i++ {
			if (week == 0 && i < startOffset) || day > daysInMonth {
				row[i] = tgmodels.InlineKeyboardButton{Text: " ", CallbackData: "cal:ignore"}
			} else {
				cellDate := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
				if cellDate.Before(todayDate) {
					// Прошедшая дата — неактивна
					row[i] = tgmodels.InlineKeyboardButton{
						Text:         fmt.Sprintf("·%d·", day),
						CallbackData: "cal:ignore",
					}
				} else {
					dateStr := fmt.Sprintf("%04d-%02d-%02d", year, int(month), day)
					row[i] = tgmodels.InlineKeyboardButton{
						Text:         fmt.Sprintf("%d", day),
						CallbackData: fmt.Sprintf("cal:day:%s", dateStr),
					}
				}
				day++
			}
		}
		rows = append(rows, row)
	}

	rows = append(rows, []tgmodels.InlineKeyboardButton{
		{Text: "❌ Отмена", CallbackData: "cal:cancel"},
	})

	return &tgmodels.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// ──────────────────────────── выбор часа ────────────────────────────

func buildHourPicker(dateStr string) *tgmodels.InlineKeyboardMarkup {
	rows := [][]tgmodels.InlineKeyboardButton{}

	// Заголовок
	rows = append(rows, []tgmodels.InlineKeyboardButton{
		{Text: fmt.Sprintf("🕐 Выберите час (%s)", dateStr), CallbackData: "cal:ignore"},
	})

	// 4 ряда по 6 часов: 0-5, 6-11, 12-17, 18-23
	for rowStart := 0; rowStart < 24; rowStart += 6 {
		row := make([]tgmodels.InlineKeyboardButton, 6)
		for i := 0; i < 6; i++ {
			h := rowStart + i
			row[i] = tgmodels.InlineKeyboardButton{
				Text:         fmt.Sprintf("%02d", h),
				CallbackData: fmt.Sprintf("cal:hour:%s:%d", dateStr, h),
			}
		}
		rows = append(rows, row)
	}

	rows = append(rows, []tgmodels.InlineKeyboardButton{
		{Text: "⬅ Назад к календарю", CallbackData: "cal:back_to_cal"},
		{Text: "❌ Отмена", CallbackData: "cal:cancel"},
	})

	return &tgmodels.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// ──────────────────────────── выбор минут ────────────────────────────

func buildMinutePicker(dateStr string, hour int) *tgmodels.InlineKeyboardMarkup {
	rows := [][]tgmodels.InlineKeyboardButton{}

	rows = append(rows, []tgmodels.InlineKeyboardButton{
		{Text: fmt.Sprintf("🕐 Выберите минуты (%s %02d:??)", dateStr, hour), CallbackData: "cal:ignore"},
	})

	// 2 ряда: 00 05 10 15 20 25 | 30 35 40 45 50 55
	for rowStart := 0; rowStart < 60; rowStart += 30 {
		row := []tgmodels.InlineKeyboardButton{}
		for m := rowStart; m < rowStart+30; m += 5 {
			row = append(row, tgmodels.InlineKeyboardButton{
				Text:         fmt.Sprintf("%02d", m),
				CallbackData: fmt.Sprintf("cal:min:%s:%d:%d", dateStr, hour, m),
			})
		}
		rows = append(rows, row)
	}

	rows = append(rows, []tgmodels.InlineKeyboardButton{
		{Text: fmt.Sprintf("⬅ Назад к часам"), CallbackData: fmt.Sprintf("cal:back_to_hours:%s", dateStr)},
		{Text: "❌ Отмена", CallbackData: "cal:cancel"},
	})

	return &tgmodels.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// ──────────────────────────── отправка / обновление ────────────────────────────

func sendCalendar(ctx context.Context, b *bot.Bot, chatID int64, eventName string, year int, month time.Month) {
	kb := buildCalendar(year, month)
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        fmt.Sprintf("📅 Выберите дату для события <b>%s</b>:", eventName),
		ParseMode:   tgmodels.ParseModeHTML,
		ReplyMarkup: kb,
	})
	if err != nil {
		logger.Errorf("Ошибка отправки календаря: %v", err)
	}
}

func editCalendar(ctx context.Context, b *bot.Bot, chatID int64, messageID int, eventName string, year int, month time.Month) {
	kb := buildCalendar(year, month)
	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        fmt.Sprintf("📅 Выберите дату для события <b>%s</b>:", eventName),
		ParseMode:   tgmodels.ParseModeHTML,
		ReplyMarkup: kb,
	})
	if err != nil {
		logger.Errorf("Ошибка обновления календаря: %v", err)
	}
}

func editToHourPicker(ctx context.Context, b *bot.Bot, chatID int64, messageID int, eventName, dateStr string) {
	kb := buildHourPicker(dateStr)
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        fmt.Sprintf("🕐 Выберите час для события <b>%s</b> (%s):", eventName, dateStr),
		ParseMode:   tgmodels.ParseModeHTML,
		ReplyMarkup: kb,
	})
}

func editToMinutePicker(ctx context.Context, b *bot.Bot, chatID int64, messageID int, eventName, dateStr string, hour int) {
	kb := buildMinutePicker(dateStr, hour)
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        fmt.Sprintf("🕐 Выберите минуты для события <b>%s</b> (%s %02d:??):", eventName, dateStr, hour),
		ParseMode:   tgmodels.ParseModeHTML,
		ReplyMarkup: kb,
	})
}

// ──────────────────────────── обработчик callback query ────────────────────────────

func handleCalendarCallback(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	cb := update.CallbackQuery
	if cb == nil {
		return
	}

	data := cb.Data
	chatID := cb.Message.Message.Chat.ID
	userID := cb.From.ID
	messageID := cb.Message.Message.ID

	defer func() {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})
	}()

	pe := getPending(chatID, userID)
	getName := func() string {
		if pe != nil {
			return pe.Name
		}
		return ""
	}

	switch {
	case data == "cal:ignore":
		return

	case data == "cal:cancel":
		deletePending(chatID, userID)
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Создание события отменено.",
		})
		return

	// ──── навигация по месяцам ────
	case strings.HasPrefix(data, "cal:prev:"):
		var year, mon int
		fmt.Sscanf(data, "cal:prev:%d:%d", &year, &mon)
		t := time.Date(year, time.Month(mon), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
		editCalendar(ctx, b, chatID, messageID, getName(), t.Year(), t.Month())
		return

	case strings.HasPrefix(data, "cal:next:"):
		var year, mon int
		fmt.Sscanf(data, "cal:next:%d:%d", &year, &mon)
		t := time.Date(year, time.Month(mon), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
		editCalendar(ctx, b, chatID, messageID, getName(), t.Year(), t.Month())
		return

	// ──── назад к календарю из выбора часов ────
	case data == "cal:back_to_cal":
		if pe != nil {
			pe.Date = ""
			pe.Hour = -1
		}
		now := time.Now()
		editCalendar(ctx, b, chatID, messageID, getName(), now.Year(), now.Month())
		return

	// ──── назад к часам из выбора минут ────
	case strings.HasPrefix(data, "cal:back_to_hours:"):
		dateStr := strings.TrimPrefix(data, "cal:back_to_hours:")
		if pe != nil {
			pe.Hour = -1
		}
		editToHourPicker(ctx, b, chatID, messageID, getName(), dateStr)
		return

	// ──── выбор дня → переход к часам ────
	case strings.HasPrefix(data, "cal:day:"):
		dateStr := strings.TrimPrefix(data, "cal:day:")
		if pe == nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: messageID,
				Text:      "⚠️ Сессия истекла. Используйте /set_date заново.",
			})
			return
		}
		pe.Date = dateStr
		pe.Hour = -1
		editToHourPicker(ctx, b, chatID, messageID, pe.Name, dateStr)
		return

	// ──── выбор часа → переход к минутам ────
	case strings.HasPrefix(data, "cal:hour:"):
		var dateStr string
		var hour int
		fmt.Sscanf(data, "cal:hour:%10s:%d", &dateStr, &hour)
		if pe == nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: messageID,
				Text:      "⚠️ Сессия истекла. Используйте /set_date заново.",
			})
			return
		}
		pe.Hour = hour
		editToMinutePicker(ctx, b, chatID, messageID, pe.Name, dateStr, hour)
		return

	// ──── выбор минут → создание события ────
	case strings.HasPrefix(data, "cal:min:"):
		var dateStr string
		var hour, minute int
		fmt.Sscanf(data, "cal:min:%10s:%d:%d", &dateStr, &hour, &minute)
		if pe == nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: messageID,
				Text:      "⚠️ Сессия истекла. Используйте /set_date заново.",
			})
			return
		}

		formattedDate := fmt.Sprintf("%s %02d:%02d", dateStr, hour, minute)

		// Создание события в БД
		if err := store.CreateEvent(ctx, chatID, pe.Name, formattedDate, pe.Description); err != nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: messageID,
				Text:      fmt.Sprintf("❌ Ошибка создания события: %s", err),
			})
			deletePending(chatID, userID)
			return
		}

		// Привязка к пользователю
		event, err := store.GetEvent(ctx, chatID, pe.Name)
		if err == nil && event != nil {
			_ = store.AddEventToUser(ctx, chatID, userID, event.ID)
		}

		logger.Infof("Событие создано через календарь: %s → %s (chat_id=%d)", pe.Name, formattedDate, chatID)

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      fmt.Sprintf("✅ Событие <b>%s</b> создано на %s!\nИспользуйте /%s для информации.", pe.Name, formattedDate, pe.Name),
			ParseMode: tgmodels.ParseModeHTML,
		})
		deletePending(chatID, userID)
	}
}
