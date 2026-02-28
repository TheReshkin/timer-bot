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

// ──────────────────────────── pending events (ожидание выбора даты) ────────────────────────────

// pendingEvent хранит данные о событии, для которого ещё не выбрана дата.
type pendingEvent struct {
	Name        string
	Description string
	ChatID      int64
	UserID      int64
}

var (
	// pendingEvents: ключ — "chatID:userID", значение — pendingEvent
	pendingEvents   = make(map[string]*pendingEvent)
	pendingEventsMu sync.Mutex
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

// ──────────────────────────── генерация inline-календаря ────────────────────────────

var russianMonths = [12]string{
	"Январь", "Февраль", "Март", "Апрель",
	"Май", "Июнь", "Июль", "Август",
	"Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
}

var shortWeekdays = [7]string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}

// buildCalendar создаёт inline-клавиатуру с календарём на заданный год/месяц.
func buildCalendar(year int, month time.Month) *tgmodels.InlineKeyboardMarkup {
	rows := [][]tgmodels.InlineKeyboardButton{}

	// Заголовок: «◀ Март 2026 ▶»
	header := []tgmodels.InlineKeyboardButton{
		{Text: "◀", CallbackData: fmt.Sprintf("cal:prev:%d:%d", year, int(month))},
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

	// Первый день месяца
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	// weekday: Monday=0 .. Sunday=6
	startOffset := int(firstDay.Weekday()+6) % 7
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()

	day := 1
	for week := 0; day <= daysInMonth; week++ {
		row := make([]tgmodels.InlineKeyboardButton, 7)
		for i := 0; i < 7; i++ {
			if (week == 0 && i < startOffset) || day > daysInMonth {
				row[i] = tgmodels.InlineKeyboardButton{Text: " ", CallbackData: "cal:ignore"}
			} else {
				dateStr := fmt.Sprintf("%04d-%02d-%02d", year, int(month), day)
				row[i] = tgmodels.InlineKeyboardButton{
					Text:         fmt.Sprintf("%d", day),
					CallbackData: fmt.Sprintf("cal:day:%s", dateStr),
				}
				day++
			}
		}
		rows = append(rows, row)
	}

	// Кнопка отмены
	rows = append(rows, []tgmodels.InlineKeyboardButton{
		{Text: "❌ Отмена", CallbackData: "cal:cancel"},
	})

	return &tgmodels.InlineKeyboardMarkup{InlineKeyboard: rows}
}

// ──────────────────────────── отправка / обновление календаря ────────────────────────────

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

	// Всегда отвечаем на callback query, чтобы убрать «часики»
	defer func() {
		b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: cb.ID})
	}()

	pe := getPending(chatID, userID)

	switch {
	case data == "cal:ignore":
		// Ничего не делаем
		return

	case data == "cal:cancel":
		deletePending(chatID, userID)
		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      "❌ Создание события отменено.",
		})
		return

	case strings.HasPrefix(data, "cal:prev:"):
		// cal:prev:YYYY:M
		var year, mon int
		fmt.Sscanf(data, "cal:prev:%d:%d", &year, &mon)
		// Предыдущий месяц
		t := time.Date(year, time.Month(mon), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
		name := ""
		if pe != nil {
			name = pe.Name
		}
		editCalendar(ctx, b, chatID, messageID, name, t.Year(), t.Month())
		return

	case strings.HasPrefix(data, "cal:next:"):
		var year, mon int
		fmt.Sscanf(data, "cal:next:%d:%d", &year, &mon)
		t := time.Date(year, time.Month(mon), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
		name := ""
		if pe != nil {
			name = pe.Name
		}
		editCalendar(ctx, b, chatID, messageID, name, t.Year(), t.Month())
		return

	case strings.HasPrefix(data, "cal:day:"):
		// cal:day:YYYY-MM-DD
		dateStr := strings.TrimPrefix(data, "cal:day:")
		if pe == nil {
			b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    chatID,
				MessageID: messageID,
				Text:      "⚠️ Сессия истекла. Используйте /set_date заново.",
			})
			return
		}

		formattedDate := dateStr + " 00:00"

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

		logger.Infof("Событие создано через календарь: %s → %s (chat_id=%d)", pe.Name, dateStr, chatID)

		b.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    chatID,
			MessageID: messageID,
			Text:      fmt.Sprintf("✅ Событие <b>%s</b> создано на %s!\nИспользуйте /%s для информации.", pe.Name, dateStr, pe.Name),
			ParseMode: tgmodels.ParseModeHTML,
		})
		deletePending(chatID, userID)
	}
}
