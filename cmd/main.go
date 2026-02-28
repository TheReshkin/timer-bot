package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/TheReshkin/timer-bot/internal/config"
	"github.com/TheReshkin/timer-bot/internal/models"
	"github.com/TheReshkin/timer-bot/internal/services"
	"github.com/TheReshkin/timer-bot/internal/storage"
	"github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

var logger *zap.Logger

func main() {
	// Инициализация логгера
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatalf("Не удалось инициализировать логгер: %v", err)
	}
	defer logger.Sync()

	err = godotenv.Load()
	if err != nil {
		logger.Error("Ошибка при загрузке .env файла", zap.Error(err))
	}

	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		logger.Fatal("TELEGRAM_TOKEN не задан")
	}

	// Инициализация бота
	b, err := bot.New(telegramToken)
	if err != nil {
		log.Fatal(err)
	}

	// Получение имени бота
	me, err := b.GetMe(context.Background())
	if err != nil {
		logger.Fatal("Не удалось получить информацию о боте", zap.Error(err))
	}
	botName := me.Username
	logger.Info("Бот инициализирован", zap.String("bot_name", botName))

	// Инициализация storage и сервисов
	store := storage.NewJSONStorage()
	eventService := services.NewEventService(store)
	userService := services.NewUserService(store)

	// Загрузка существующих команд
	loadExistingCommands(b, eventService)

	// Регистрация команд
	b.RegisterHandler(bot.HandlerTypeMessageText, "/set_date", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleSetDate(ctx, b, update, eventService, userService)
	})
	b.RegisterHandler(bot.HandlerTypeMessageText, "/list", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleList(ctx, b, update, eventService)
	})
	b.RegisterHandler(bot.HandlerTypeMessageText, "/all", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleAll(ctx, b, update, eventService)
	})
	b.RegisterHandler(bot.HandlerTypeMessageText, "/active", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleActive(ctx, b, update, eventService)
	})
	b.RegisterHandler(bot.HandlerTypeMessageText, "/outdated", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleOutdated(ctx, b, update, eventService)
	})
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleHelp(ctx, b, update)
	})

	// Обработчик для динамических команд - регистрируем последним
	b.RegisterHandler(bot.HandlerTypeMessageText, "/", bot.MatchTypePrefix, func(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
		handleDynamicOrUnknown(ctx, b, update, eventService)
	})

	// Запуск бота
	logger.Info("Бот запущен")
	b.Start(context.Background())
}

// normalizeCommand удаляет суффикс @bot_username из команды
func normalizeCommand(text string) string {
	if strings.Contains(text, "@") {
		parts := strings.Split(text, "@")
		return parts[0]
	}
	return text
}

func handleSetDate(ctx context.Context, b *bot.Bot, update *tgmodels.Update, eventService *services.EventService, userService *services.UserService) {
	if update.Message == nil {
		return
	}

	// Нормализуем команду
	command := normalizeCommand(update.Message.Text)
	if !strings.HasPrefix(command, "/set_date") {
		return // Не наша команда
	}

	parts := strings.Fields(command)
	if len(parts) < 3 {
		sendMessage(ctx, b, update.Message.Chat.ID, "Используйте формат:\n/set_date YYYY-MM-DD HH:MM event_name [description]\n/set_date YYYY-MM-DD event_name [description]\n/set_date DD.MM.YYYY event_name [description]")
		return
	}

	var dateStr, name, description string

	// Проверяем, является ли вторая часть времени (HH:MM)
	if len(parts) >= 4 && regexp.MustCompile(`^\d{1,2}:\d{2}$`).MatchString(parts[2]) {
		// Формат: /set_date YYYY-MM-DD HH:MM name [desc]
		dateStr = parts[1] + " " + parts[2]
		name = parts[3]
		if len(parts) > 4 {
			description = strings.Join(parts[4:], " ")
		}
	} else {
		// Формат: /set_date YYYY-MM-DD name [desc] или /set_date DD.MM.YYYY name [desc]
		dateStr = parts[1]
		name = parts[2]
		if len(parts) > 3 {
			description = strings.Join(parts[3:], " ")
		}
	}

	// Парсинг и валидация даты
	parsedDate, err := models.ParseEventDate(dateStr)
	if err != nil {
		sendMessage(ctx, b, update.Message.Chat.ID, fmt.Sprintf("Ошибка парсинга даты: %s", err.Error()))
		return
	}

	// Преобразование в новый формат для хранения
	formattedDate := models.FormatEventDate(parsedDate)

	err = eventService.CreateEvent(update.Message.Chat.ID, name, formattedDate, description)
	if err != nil {
		sendMessage(ctx, b, update.Message.Chat.ID, fmt.Sprintf("Ошибка: %s", err.Error()))
		return
	}

	// Регистрация динамической команды
	registerDynamicCommand(b, eventService, name)

	// Добавление события к пользователю
	event, _ := eventService.GetEvent(update.Message.Chat.ID, name)
	if event != nil {
		userService.AddEventToUser(update.Message.Chat.ID, update.Message.From.ID, *event)
	}

	sendMessage(ctx, b, update.Message.Chat.ID, fmt.Sprintf("Событие '%s' добавлено! Используйте /%s для информации.", name, name))
}

func handleList(ctx context.Context, b *bot.Bot, update *tgmodels.Update, eventService *services.EventService) {
	if update.Message == nil {
		return
	}

	// Нормализуем команду
	command := normalizeCommand(update.Message.Text)
	if command != "/list" {
		return // Не наша команда
	}

	// Получаем события из текущего чата
	events, err := eventService.ListEvents(update.Message.Chat.ID)
	if err != nil {
		sendMessage(ctx, b, update.Message.Chat.ID, "Ошибка при получении событий")
		return
	}

	// Если это не тестовый чат, добавляем события из тестового чата
	testChatID := config.LoadTestChatID()
	if update.Message.Chat.ID != testChatID {
		testEvents, err := eventService.ListEvents(testChatID)
		if err == nil {
			events = append(events, testEvents...)
		}
	}

	if len(events) == 0 {
		sendMessage(ctx, b, update.Message.Chat.ID, "Нет событий")
		return
	}

	message := "События:\n"
	for _, event := range events {
		message += fmt.Sprintf("- %s: %s (команда /%s)\n", event.Name, event.Date, event.Name)
	}
	sendMessage(ctx, b, update.Message.Chat.ID, message)
}

func handleAll(ctx context.Context, b *bot.Bot, update *tgmodels.Update, eventService *services.EventService) {
	handleList(ctx, b, update, eventService) // Показывает все события из текущего и тестового чатов
}

func handleActive(ctx context.Context, b *bot.Bot, update *tgmodels.Update, eventService *services.EventService) {
	if update.Message == nil {
		return
	}

	// Нормализуем команду
	command := normalizeCommand(update.Message.Text)
	if command != "/active" {
		return // Не наша команда
	}

	// Получаем события из текущего чата
	events, err := eventService.ListEvents(update.Message.Chat.ID)
	if err != nil {
		sendMessage(ctx, b, update.Message.Chat.ID, "Ошибка при получении событий")
		return
	}

	// Если это не тестовый чат, добавляем события из тестового чата
	testChatID := config.LoadTestChatID()
	if update.Message.Chat.ID != testChatID {
		testEvents, err := eventService.ListEvents(testChatID)
		if err == nil {
			events = append(events, testEvents...)
		}
	}

	activeEvents := []models.Event{}
	for _, event := range events {
		if event.Status == models.StatusActive {
			activeEvents = append(activeEvents, event)
		}
	}

	if len(activeEvents) == 0 {
		sendMessage(ctx, b, update.Message.Chat.ID, "Нет активных событий")
		return
	}

	message := "Активные события:\n"
	for _, event := range activeEvents {
		message += fmt.Sprintf("- %s: %s\n", event.Name, event.Date)
	}
	sendMessage(ctx, b, update.Message.Chat.ID, message)
}

func handleOutdated(ctx context.Context, b *bot.Bot, update *tgmodels.Update, eventService *services.EventService) {
	if update.Message == nil {
		return
	}

	// Нормализуем команду
	command := normalizeCommand(update.Message.Text)
	if command != "/outdated" {
		return // Не наша команда
	}

	// Получаем события из текущего чата
	events, err := eventService.ListEvents(update.Message.Chat.ID)
	if err != nil {
		sendMessage(ctx, b, update.Message.Chat.ID, "Ошибка при получении событий")
		return
	}

	// Если это не тестовый чат, добавляем события из тестового чата
	testChatID := config.LoadTestChatID()
	if update.Message.Chat.ID != testChatID {
		testEvents, err := eventService.ListEvents(testChatID)
		if err == nil {
			events = append(events, testEvents...)
		}
	}

	outdatedEvents := []models.Event{}
	for _, event := range events {
		if event.Status == models.StatusOutdated {
			outdatedEvents = append(outdatedEvents, event)
		}
	}

	if len(outdatedEvents) == 0 {
		sendMessage(ctx, b, update.Message.Chat.ID, "Нет устаревших событий")
		return
	}

	message := "Устаревшие события:\n"
	for _, event := range outdatedEvents {
		message += fmt.Sprintf("- %s: %s\n", event.Name, event.Date)
	}
	sendMessage(ctx, b, update.Message.Chat.ID, message)
}

func handleHelp(ctx context.Context, b *bot.Bot, update *tgmodels.Update) {
	if update.Message == nil {
		return
	}

	// Нормализуем команду
	command := normalizeCommand(update.Message.Text)
	if command != "/help" {
		return // Не наша команда
	}

	helpText := `Команды:
/set_date YYYY-MM-DD HH:MM event_name [description] - добавить событие с временем
/set_date YYYY-MM-DD event_name [description] - добавить событие (время 00:00)
/set_date DD.MM.YYYY event_name [description] - добавить событие (старый формат)
/list - список событий
/all - все события
/active - активные события
/outdated - устаревшие события
/help - справка
/event_name - информация о событии`
	sendMessage(ctx, b, update.Message.Chat.ID, helpText)
}

func handleDynamicOrUnknown(ctx context.Context, b *bot.Bot, update *tgmodels.Update, eventService *services.EventService) {
	if update.Message == nil {
		logger.Debug("Получено обновление без сообщения")
		return
	}
	logger.Info("Получено сообщение", zap.String("text", update.Message.Text))
	if !strings.HasPrefix(update.Message.Text, "/") {
		logger.Debug("Сообщение не является командой")
		return
	}
	command := strings.TrimPrefix(update.Message.Text, "/")
	if strings.Contains(command, "@") {
		parts := strings.Split(command, "@")
		command = parts[0]
	}

	// Проверяем, является ли команда системной
	systemCommands := []string{"set_date", "list", "all", "active", "outdated", "help", "start"}
	for _, sysCmd := range systemCommands {
		if command == sysCmd {
			logger.Debug("Системная команда, пропускаем", zap.String("command", command))
			return
		}
	}

	logger.Info("Обработка динамической команды", zap.String("command", command))
	handleDynamicCommand(ctx, b, update, command, eventService)
}

func handleDynamicCommand(ctx context.Context, b *bot.Bot, update *tgmodels.Update, name string, eventService *services.EventService) {
	if update.Message == nil {
		return
	}

	logger.Info("Поиск события",
		zap.String("event_name", name),
		zap.Int64("chat_id", update.Message.Chat.ID))

	// Сначала ищем в текущем чате
	event, err := eventService.GetEvent(update.Message.Chat.ID, name)
	if err != nil {
		// Если не найдено в текущем чате, ищем в других чатах
		logger.Info("Событие не найдено в текущем чате, ищем в других чатах",
			zap.String("event_name", name))
		var foundChatID int64
		event, foundChatID, err = eventService.FindEventAcrossChats(name, update.Message.Chat.ID)
		if err == nil {
			logger.Info("Событие найдено в другом чате",
				zap.String("event_name", name),
				zap.Int64("found_in_chat_id", foundChatID))
		}
	}

	if err != nil {
		logger.Warn("Событие не найдено",
			zap.String("event_name", name),
			zap.Error(err))
		sendMessage(ctx, b, update.Message.Chat.ID, fmt.Sprintf("Событие '%s' не найдено", name))
		return
	}

	logger.Info("Найдено событие",
		zap.String("event_name", event.Name),
		zap.String("date", event.Date))

	// Обновление статуса события в его чате
	eventService.UpdateEventStatus(event.ChatID, name)

	// Расчёт времени до события
	parsedDate, err := models.ParseEventDate(event.Date)
	if err != nil {
		logger.Error("Ошибка парсинга даты события", zap.Error(err))
		sendMessage(ctx, b, update.Message.Chat.ID, "Ошибка при расчете времени")
		return
	}

	duration := time.Until(parsedDate)
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	message := fmt.Sprintf("Событие: %s\nДата: %s\n", event.Name, event.Date)
	if event.Description != "" {
		message += fmt.Sprintf("Описание: %s\n", event.Description)
	}
	if duration > 0 {
		message += fmt.Sprintf("Осталось: %d дней, %d часов, %d минут", days, hours, minutes)
	} else {
		message += "Событие прошло"
	}

	sendMessage(ctx, b, update.Message.Chat.ID, message)
}

func sendMessage(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
}

func loadExistingCommands(b *bot.Bot, eventService *services.EventService) {
	commands := []tgmodels.BotCommand{
		{Command: "set_date", Description: "Добавить событие (/set_date DD.MM.YYYY name)"},
		{Command: "list", Description: "Список событий"},
		{Command: "all", Description: "Все события"},
		{Command: "active", Description: "Активные события"},
		{Command: "outdated", Description: "Устаревшие события"},
		{Command: "help", Description: "Справка"},
	}

	logger.Info("Устанавливаем базовые команды", zap.Int("count", len(commands)))
	_, err := b.SetMyCommands(context.Background(), &bot.SetMyCommandsParams{
		Commands: commands,
	})
	if err != nil {
		logger.Error("Ошибка при установке команд", zap.Error(err))
	} else {
		logger.Info("Команды успешно установлены")
	}
}

func registerDynamicCommand(b *bot.Bot, eventService *services.EventService, name string) {
	logger.Info("Динамическая команда зарегистрирована локально",
		zap.String("event_name", name))
	// Больше не регистрируем глобальные команды - обрабатываем динамически
}
