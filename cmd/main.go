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

	// Инициализация бота
	b, err := bot.New(cfg.Token)
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
	if len(parts) < 3 {
		sendMessage(ctx, b, update.Message.Chat.ID,
			"Используйте формат:\n"+
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
		sendMessage(ctx, b, update.Message.Chat.ID, fmt.Sprintf("Ошибка парсинга даты: %s", err))
		return
	}
	formattedDate := parsedDate.Format("2006-01-02 15:04")

	// Создание события в БД
	if err := store.CreateEvent(ctx, update.Message.Chat.ID, name, formattedDate, description); err != nil {
		sendMessage(ctx, b, update.Message.Chat.ID, fmt.Sprintf("Ошибка: %s", err))
		return
	}

	// Привязка события к пользователю
	event, err := store.GetEvent(ctx, update.Message.Chat.ID, name)
	if err == nil && event != nil {
		_ = store.AddEventToUser(ctx, update.Message.Chat.ID, update.Message.From.ID, event.ID)
	}

	logger.Infof("Событие создано: %s (chat_id=%d)", name, update.Message.Chat.ID)
	sendMessage(ctx, b, update.Message.Chat.ID,
		fmt.Sprintf("Событие '%s' добавлено! Используйте /%s для информации.", name, name))
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
/set_date YYYY-MM-DD HH:MM event_name [description] — добавить событие с временем
/set_date YYYY-MM-DD event_name [description] — добавить событие (время 00:00)
/set_date DD.MM.YYYY event_name [description] — добавить событие (формат DD.MM)
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
	commands := []tgmodels.BotCommand{
		{Command: "set_date", Description: "Добавить событие"},
		{Command: "list", Description: "Список событий"},
		{Command: "active", Description: "Активные события"},
		{Command: "outdated", Description: "Устаревшие события"},
		{Command: "help", Description: "Справка"},
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
