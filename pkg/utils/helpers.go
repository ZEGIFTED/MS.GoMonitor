package utils

import (
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"time"
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
