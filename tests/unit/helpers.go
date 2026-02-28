package unit

import (
	"testing"
	"time"
)

// getTestLocation returns the Moscow timezone for testing
func getTestLocation(t *testing.T) *time.Location {
	t.Helper()
	location, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatalf("Failed to load test timezone: %v", err)
	}
	return location
}
