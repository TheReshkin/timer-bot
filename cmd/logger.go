package main

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/TheReshkin/timer-bot/internal/config"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Set log level from LOG_LEVEL env var (default: info)
	lvl := strings.ToLower(os.Getenv("LOG_LEVEL"))
	if lvl == "" {
		logger.SetLevel(logrus.InfoLevel)
	} else {
		parsed, err := logrus.ParseLevel(lvl)
		if err != nil {
			logger.SetLevel(logrus.InfoLevel)
			logger.Warnf("invalid LOG_LEVEL %q, defaulting to info", lvl)
		} else {
			logger.SetLevel(parsed)
		}
	}

	logger.SetOutput(os.Stdout)
}

// WithComponent возвращает логгер с полем "component" (и "service" из конфига) для структурированного логирования.
// Использование: WithComponent("trades").WithField("operation", "sync").Info("...")
func WithComponent(component string) *logrus.Entry {
	entry := logger.WithField("component", component)
	if c := config.GetConfig(); c != nil && c.ServiceName != "" {
		entry = entry.WithField("service", c.ServiceName)
	}
	return entry
}

// LogSyncResult пишет структурированный лог результата синхронизации (успех или ошибка).
func LogSyncResult(log *logrus.Entry, operation string, durationMs int64, items int, err error) {
	fields := logrus.Fields{
		"operation":   operation,
		"duration_ms": durationMs,
		"items":       items,
	}
	if err != nil {
		log.WithError(err).WithFields(fields).Error("sync failed")
		return
	}
	if items > 0 {
		log.WithFields(fields).Debug("sync completed")
	}
}
