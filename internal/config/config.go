package config

import (
	"log"
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

type Config struct {
	Token       string
	AdminID     int
	TestChatID  int
	ServiceName string
	DatabaseURL string
}

// // TestChatID is the hardcoded chat ID used for testing and fallback searches
// var TestChatID int64 = 332288278

var (
	configInstance *Config
	configOnce     sync.Once
)

func LoadConfig() *Config {
	configOnce.Do(func() {
		err := godotenv.Load()
		if err != nil {
			log.Fatalf("Ошибка при загрузке .env файла: %v", err)
		}

		configInstance = &Config{
			Token:       os.Getenv("TELEGRAM_TOKEN"),
			AdminID:     getEnvAsInt("ADMIN_ID", 0),
			TestChatID:  getEnvAsInt("TEST_CHAT_ID", 0),
			ServiceName: os.Getenv("c"),
			DatabaseURL: os.Getenv("DATABASE_URL"),
		}
	})
	return configInstance
}

// GetConfig returns the current config singleton (nil if not yet loaded).
func GetConfig() *Config {
	return configInstance
}

// // LoadTestChatID loads the test chat ID from environment variable or uses default
// func LoadTestChatID() int64 {
// 	if envValue := os.Getenv("TEST_CHAT_ID"); envValue != "" {
// 		if parsed, err := strconv.ParseInt(envValue, 10, 64); err == nil {
// 			return parsed
// 		}
// 	}
// 	return TestChatID
// }

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
