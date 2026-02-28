package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/TheReshkin/timer-bot/internal/config"
	"github.com/TheReshkin/timer-bot/internal/storage"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

// store — глобальная ссылка на PostgreSQL хранилище
var store *storage.PostgresStorage

// handleEditedMessage обрабатывает отредактированные сообщения,
// перенаправляя их в соответствующие обработчики команд.
// Также обрабатывает обычные сообщения от пользователей в режиме ожидания ввода.
func handleEditedMessage(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	// Обработка обычных сообщений, не пойманных зарегистрированными обработчиками
	// (например, ввод названия события после /set_date)
	if update.Message != nil && update.EditedMessage == nil {
		text := update.Message.Text
		if text == "" {
			return
		}
		chatID := update.Message.Chat.ID
		userID := update.Message.From.ID
		if isAwaitingName(chatID, userID) {
			handleEventNameReply(ctx, b, update)
		}
		return
	}

	if update.EditedMessage == nil {
		return
	}
	logger.Debugf("Обработка отредактированного сообщения (chat_id=%d): %s",
		update.EditedMessage.Chat.ID, update.EditedMessage.Text)

	// Подменяем Message, чтобы обработчики работали как обычно
	update.Message = update.EditedMessage

	text := update.Message.Text
	if text == "" {
		return
	}

	// Проверяем, ожидает ли пользователь ввод названия события
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	if isAwaitingName(chatID, userID) {
		handleEventNameReply(ctx, b, update)
		return
	}

	cmd := normalizeCommand(text)
	switch {
	case strings.HasPrefix(cmd, "/set_date"):
		handleSetDate(ctx, b, update)
	case cmd == "/list" || cmd == "/all":
		handleList(ctx, b, update)
	case cmd == "/active":
		handleActive(ctx, b, update)
	case cmd == "/outdated":
		handleOutdated(ctx, b, update)
	case cmd == "/help":
		handleHelp(ctx, b, update)
	case strings.HasPrefix(cmd, "/"):
		handleDynamicOrUnknown(ctx, b, update)
	}
}

func main() {
	// Загрузка конфигурации (читает .env внутри)
	cfg := config.LoadConfig()

	if cfg.Token == "" {
		logger.Fatal("TELEGRAM_TOKEN не задан")
	}

	// Подключение к БД
	store = storage.NewPostgresStorage(cfg.DatabaseURL)
	defer store.Close()
	logger.Info("PostgreSQL подключён")

	// Инициализация бота с default handler для отредактированных сообщений
	b, err := bot.New(cfg.Token,
		bot.WithAllowedUpdates(bot.AllowedUpdates{"message", "edited_message", "callback_query"}),
		bot.WithDefaultHandler(handleEditedMessage),
	)
	if err != nil {
		logger.Fatalf("Ошибка создания бота: %v", err)
	}

	// Получение имени бота
	me, err := b.GetMe(context.Background())
	if err != nil {
		logger.Fatalf("Не удалось получить информацию о боте: %v", err)
	}
	logger.Infof("Бот инициализирован: @%s", me.Username)

	// Установка команд в меню Telegram
	loadExistingCommands(b)

	// Регистрация обработчиков команд
	b.RegisterHandler(bot.HandlerTypeMessageText, "/set_date", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleSetDate(ctx, b, update)
	})
	b.RegisterHandler(bot.HandlerTypeMessageText, "/list", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleList(ctx, b, update)
	})
	b.RegisterHandler(bot.HandlerTypeMessageText, "/active", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleActive(ctx, b, update)
	})
	b.RegisterHandler(bot.HandlerTypeMessageText, "/outdated", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleOutdated(ctx, b, update)
	})
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleHelp(ctx, b, update)
	})

	// Обработчик callback query для inline-календаря
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "cal:", bot.MatchTypePrefix, handleCalendarCallback)

	// Обработчик для динамических команд — регистрируем последним
	b.RegisterHandler(bot.HandlerTypeMessageText, "/", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleDynamicOrUnknown(ctx, b, update)
	})

	// Запуск бота
	logger.Info("Бот запущен")
	b.Start(context.Background())
}

// ──────────────────────────── утилиты ────────────────────────────

// normalizeCommand удаляет суффикс @bot_username из команды
func normalizeCommand(text string) string {
	if idx := strings.Index(text, "@"); idx != -1 {
		return text[:idx]
	}
	return text
}

// parseEventDate парсит дату в форматах "YYYY-MM-DD HH:MM", "YYYY-MM-DD", "DD.MM.YYYY"
func parseEventDate(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02 15:04",
		"2006-01-02",
		"02.01.2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("неизвестный формат даты: %s", s)
}

// looksLikeDate проверяет, похожа ли строка на дату (начинается с цифры).
func looksLikeDate(s string) bool {
	if len(s) == 0 {
		return false
	}
	return s[0] >= '0' && s[0] <= '9'
}

func sendMessage(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		logger.Errorf("Ошибка отправки сообщения chat_id=%d: %v", chatID, err)
	}
}

// ──────────────────────────── обработчики ────────────────────────────

func handleSetDate(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if update.Message == nil {
		return
	}
	command := normalizeCommand(update.Message.Text)
	if !strings.HasPrefix(command, "/set_date") {
		return
	}

	parts := strings.Fields(command)
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	// Режим 0: /set_date (без аргументов) → запрос названия события
	if len(parts) == 1 {
		setAwaitingName(chatID, userID)
		sendMessage(ctx, b, chatID, "📝 Введите название события:")
		return
	}

	// Режим 1: /set_date <name> [description] → интерактивный календарь
	if len(parts) >= 2 && !looksLikeDate(parts[1]) {
		name := parts[1]
		description := ""
		if len(parts) > 2 {
			description = strings.Join(parts[2:], " ")
		}

		setPending(chatID, userID, &pendingEvent{
			Name:        name,
			Description: description,
			ChatID:      chatID,
			UserID:      userID,
			Hour:        -1,
		})

		now := time.Now()
		sendCalendar(ctx, b, chatID, name, now.Year(), now.Month())
		return
	}

	// Режим 2: /set_date <date> [time] <name> [description] → прямое создание
	if len(parts) < 3 {
		sendMessage(ctx, b, chatID,
			"Используйте формат:\n"+
				"/set_date event_name [description] — интерактивный календарь\n"+
				"/set_date YYYY-MM-DD HH:MM event_name [description]\n"+
				"/set_date YYYY-MM-DD event_name [description]\n"+
				"/set_date DD.MM.YYYY event_name [description]")
		return
	}

	var dateStr, name, description string

	// Проверяем, является ли третий аргумент временем (HH:MM)
	if len(parts) >= 4 && regexp.MustCompile(`^\d{1,2}:\d{2}$`).MatchString(parts[2]) {
		dateStr = parts[1] + " " + parts[2]
		name = parts[3]
		if len(parts) > 4 {
			description = strings.Join(parts[4:], " ")
		}
	} else {
		dateStr = parts[1]
		name = parts[2]
		if len(parts) > 3 {
			description = strings.Join(parts[3:], " ")
		}
	}

	// Валидация даты
	parsedDate, err := parseEventDate(dateStr)
	if err != nil {
		sendMessage(ctx, b, chatID, fmt.Sprintf("Ошибка парсинга даты: %s", err))
		return
	}
	formattedDate := parsedDate.Format("2006-01-02 15:04")

	// Создание события в БД
	if err := store.CreateEvent(ctx, chatID, name, formattedDate, description); err != nil {
		sendMessage(ctx, b, chatID, fmt.Sprintf("Ошибка: %s", err))
		return
	}

	// Привязка события к пользователю
	event, err := store.GetEvent(ctx, chatID, name)
	if err == nil && event != nil {
		_ = store.AddEventToUser(ctx, chatID, userID, event.ID)
	}

	logger.Infof("Событие создано: %s (chat_id=%d)", name, chatID)
	sendMessage(ctx, b, chatID,
		fmt.Sprintf("Событие '%s' добавлено! Используйте /%s для информации.", name, name))
}

// handleEventNameReply обрабатывает текстовый ввод названия события
// после того, как пользователь вызвал /set_date без аргументов.
func handleEventNameReply(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	text := strings.TrimSpace(update.Message.Text)

	// Если пользователь передумал
	if text == "" || text == "/cancel" {
		clearAwaitingName(chatID, userID)
		sendMessage(ctx, b, chatID, "❌ Создание события отменено.")
		return
	}

	// Если ввели другую команду — сбрасываем ожидание
	if strings.HasPrefix(text, "/") {
		clearAwaitingName(chatID, userID)
		return
	}

	clearAwaitingName(chatID, userID)

	// Первое слово — название, остальное — описание
	parts := strings.Fields(text)
	name := parts[0]
	description := ""
	if len(parts) > 1 {
		description = strings.Join(parts[1:], " ")
	}

	setPending(chatID, userID, &pendingEvent{
		Name:        name,
		Description: description,
		ChatID:      chatID,
		UserID:      userID,
		Hour:        -1,
	})

	now := time.Now()
	sendCalendar(ctx, b, chatID, name, now.Year(), now.Month())
}

func handleList(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if update.Message == nil {
		return
	}
	command := normalizeCommand(update.Message.Text)
	if command != "/list" && command != "/all" {
		return
	}

	chatID := update.Message.Chat.ID
	events, err := store.ListEvents(ctx, chatID)
	if err != nil {
		sendMessage(ctx, b, chatID, "Ошибка при получении событий")
		return
	}

	// Добавляем события из тестового чата, если мы не в нём
	cfg := config.GetConfig()
	testChatID := int64(cfg.TestChatID)
	if chatID != testChatID {
		testEvents, err := store.ListEvents(ctx, testChatID)
		if err == nil {
			events = append(events, testEvents...)
		}
	}

	if len(events) == 0 {
		sendMessage(ctx, b, chatID, "Нет событий")
		return
	}

	msg := "События:\n"
	for _, e := range events {
		msg += fmt.Sprintf("- %s: %s (/%s)\n", e.Name, e.Date, e.Name)
	}
	sendMessage(ctx, b, chatID, msg)
}

func handleActive(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if update.Message == nil {
		return
	}
	if normalizeCommand(update.Message.Text) != "/active" {
		return
	}

	chatID := update.Message.Chat.ID
	events, err := store.ListEvents(ctx, chatID)
	if err != nil {
		sendMessage(ctx, b, chatID, "Ошибка при получении событий")
		return
	}

	cfg := config.GetConfig()
	testChatID := int64(cfg.TestChatID)
	if chatID != testChatID {
		testEvents, err := store.ListEvents(ctx, testChatID)
		if err == nil {
			events = append(events, testEvents...)
		}
	}

	var active []storage.Event
	for _, e := range events {
		if e.Status == "active" {
			active = append(active, e)
		}
	}

	if len(active) == 0 {
		sendMessage(ctx, b, chatID, "Нет активных событий")
		return
	}

	msg := "Активные события:\n"
	for _, e := range active {
		msg += fmt.Sprintf("- %s: %s\n", e.Name, e.Date)
	}
	sendMessage(ctx, b, chatID, msg)
}

func handleOutdated(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if update.Message == nil {
		return
	}
	if normalizeCommand(update.Message.Text) != "/outdated" {
		return
	}

	chatID := update.Message.Chat.ID
	events, err := store.ListEvents(ctx, chatID)
	if err != nil {
		sendMessage(ctx, b, chatID, "Ошибка при получении событий")
		return
	}

	cfg := config.GetConfig()
	testChatID := int64(cfg.TestChatID)
	if chatID != testChatID {
		testEvents, err := store.ListEvents(ctx, testChatID)
		if err == nil {
			events = append(events, testEvents...)
		}
	}

	var outdated []storage.Event
	for _, e := range events {
		if e.Status == "outdated" {
			outdated = append(outdated, e)
		}
	}

	if len(outdated) == 0 {
		sendMessage(ctx, b, chatID, "Нет устаревших событий")
		return
	}

	msg := "Устаревшие события:\n"
	for _, e := range outdated {
		msg += fmt.Sprintf("- %s: %s\n", e.Name, e.Date)
	}
	sendMessage(ctx, b, chatID, msg)
}

func handleHelp(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if update.Message == nil {
		return
	}
	if normalizeCommand(update.Message.Text) != "/help" {
		return
	}

	helpText := `Команды:
/set_date event_name [description] — добавить событие (📅 календарь)
/set_date YYYY-MM-DD event_name [description] — добавить событие напрямую
/set_date YYYY-MM-DD HH:MM event_name [description] — с указанием времени
/list — список всех событий
/active — активные события
/outdated — устаревшие события
/help — справка
/<event_name> — информация о событии`
	sendMessage(ctx, b, update.Message.Chat.ID, helpText)
}

func handleDynamicOrUnknown(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if update.Message == nil {
		return
	}
	if !strings.HasPrefix(update.Message.Text, "/") {
		return
	}

	command := strings.TrimPrefix(update.Message.Text, "/")
	if idx := strings.Index(command, "@"); idx != -1 {
		command = command[:idx]
	}

	// Пропускаем системные команды
	systemCommands := []string{"set_date", "list", "all", "active", "outdated", "help", "start"}
	for _, sc := range systemCommands {
		if command == sc {
			return
		}
	}

	logger.Debugf("Динамическая команда: %s", command)
	handleDynamicCommand(ctx, b, update, command)
}

func handleDynamicCommand(ctx context.Context, b *bot.Bot, update *tgmodels.Update, name string) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID

	// Сначала ищем в текущем чате
	event, err := store.GetEvent(ctx, chatID, name)
	if err != nil {
		// Ищем во всех остальных чатах
		event, _, err = store.FindEventAcrossChats(ctx, name, chatID)
	}

	if err != nil {
		logger.Debugf("Событие '%s' не найдено: %v", name, err)
		sendMessage(ctx, b, chatID, fmt.Sprintf("Событие '%s' не найдено", name))
		return
	}

	// Автообновление статуса: если дата прошла — помечаем outdated
	parsedDate, err := parseEventDate(event.Date)
	if err != nil {
		logger.Errorf("Ошибка парсинга даты события '%s': %v", name, err)
		sendMessage(ctx, b, chatID, "Ошибка при расчете времени")
		return
	}

	if time.Now().After(parsedDate) && event.Status != "outdated" {
		_ = store.UpdateEventStatus(ctx, event.ChatID, name, "outdated")
		event.Status = "outdated"
	}

	duration := time.Until(parsedDate)
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	msg := fmt.Sprintf("Событие: %s\nДата: %s\n", event.Name, event.Date)
	if event.Description != "" {
		msg += fmt.Sprintf("Описание: %s\n", event.Description)
	}
	if duration > 0 {
		msg += fmt.Sprintf("Осталось: %d дней, %d часов, %d минут", days, hours, minutes)
	} else {
		msg += "Событие уже прошло"
	}

	sendMessage(ctx, b, chatID, msg)
}

// ──────────────────────────── bootstrap ────────────────────────────

func loadExistingCommands(b *bot.Bot) {
	// Удаляем старые команды, чтобы Telegram точно обновил меню
	b.DeleteMyCommands(context.Background(), &bot.DeleteMyCommandsParams{})

	commands := []tgmodels.BotCommand{
		{Command: "set_date", Description: "📅 Добавить событие (календарь или дата)"},
		{Command: "list", Description: "📋 Список всех событий"},
		{Command: "active", Description: "✅ Активные события"},
		{Command: "outdated", Description: "⏰ Устаревшие события"},
		{Command: "help", Description: "❓ Справка по командам"},
	}

	_, err := b.SetMyCommands(context.Background(), &bot.SetMyCommandsParams{
		Commands: commands,
	})
	if err != nil {
		logger.Errorf("Ошибка при установке команд: %v", err)
	} else {
		logger.Infof("Команды установлены (%d)", len(commands))
	}
}
