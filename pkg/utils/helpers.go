package utils

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// IsValidCron checks if a given string is a valid cron expression
func IsValidCron(expr string) bool {
	_, err := cron.ParseStandard(expr) // Uses the standard 5-field cron format
	return err == nil
}

func IsValidUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

// ConvertUnixToTime converts a Unix timestamp (in seconds or milliseconds) to time.Time.
func ConvertUnixToTime(timestamp float64) time.Time {
	// Check if the timestamp is in milliseconds (e.g., 1736563130369)
	if timestamp > 1e12 { // Assume it's in milliseconds
		return time.Unix(0, int64(timestamp*1e6))
	}

	//Otherwise, assume it's in seconds (e.g., 1739913427.086267)
	return time.Unix(int64(timestamp), 0)

	//if timestamp > 1e12 { // If the timestamp is greater than 1e12, it's likely in milliseconds
	//	return time.Unix(0, int64(timestamp)*int64(time.Millisecond))
	//}

	//return time.Unix(int64(timestamp), int64((timestamp-float64(int64(timestamp)))*1e9))
}

// GetServiceDownTime calculates the downtime duration string since the given datetime.
// If the datetime is invalid (zero or before 1753-01-01 or after 9999-12-31), it defaults to 24 hours ago.
func GetServiceDownTime(input time.Time) string {
	// Define valid datetime range
	minDate := time.Date(1753, 1, 1, 0, 0, 0, 0, time.UTC)
	maxDate := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)

	now := time.Now().UTC()

	// If the input is zero or out of range, use 24 hours ago
	if input.IsZero() || input.Before(minDate) || input.After(maxDate) {
		input = now.Add(-24 * time.Hour)
	}

	// Calculate the difference
	duration := now.Sub(input)
	totalMinutes := int(duration.Minutes())

	days := totalMinutes / (60 * 24)
	hours := (totalMinutes % (60 * 24)) / 60
	minutes := totalMinutes % 60

	return fmt.Sprintf("%dD %dH %dM", days, hours, minutes)
}
