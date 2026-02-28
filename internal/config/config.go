package config

import (
	"os"
	"strconv"
)

// TestChatID is the hardcoded chat ID used for testing and fallback searches
var TestChatID int64 = 332288278

// LoadTestChatID loads the test chat ID from environment variable or uses default
func LoadTestChatID() int64 {
	if envValue := os.Getenv("TEST_CHAT_ID"); envValue != "" {
		if parsed, err := strconv.ParseInt(envValue, 10, 64); err == nil {
			return parsed
		}
	}
	return TestChatID
}
