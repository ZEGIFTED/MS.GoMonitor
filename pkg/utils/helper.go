package utils

import (
	"github.com/robfig/cron/v3"
	"os"
)

// GetEnvWithDefault retrieves an environment variable with a fallback default value
func GetEnvWithDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// IsValidCron checks if a given string is a valid cron expression
func IsValidCron(expr string) bool {
	_, err := cron.ParseStandard(expr) // Uses the standard 5-field cron format
	return err == nil
}
