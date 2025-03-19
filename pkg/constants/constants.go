package constants

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
		log.Printf("Warning: .env file not found. Using system environment variables")
	}
}

// GetEnvWithDefault retrieves an environment variable with a fallback default value
func GetEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

var DB = DBConfig{
	Host:     GetEnvWithDefault("DB_HOST", "localhost"),
	Port:     GetEnvWithDefault("DB_PORT", "1433"),
	Name:     GetEnvWithDefault("DB_NAME", "MS"),
	User:     GetEnvWithDefault("DB_USER", "sa"),
	Password: GetEnvWithDefault("DB_PASSWORD", "dbuser123$"),
}

var (
	LogPath            = "logs/"
	LogFileName        = LogPath + "ms-svc_monitor.log"
	DatabaseConnString = fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s;", DB.Host, DB.User, DB.Password, DB.Port, DB.Name)
	SMTPHost           = GetEnvWithDefault("MAIL_HOST", "localhost")
	SMTPPort           = GetEnvWithDefault("MAIL_PORT", "25")
	SMTPUser           = GetEnvWithDefault("MAIL_USER", "test-notification@nibss-plc.com.ng")
	SMTPPass           = GetEnvWithDefault("MAIL_PASS", "password123$_")
	STMP_ADMIN_MAIL    = GetEnvWithDefault("STMP_ADMIN_MAIL", "test-notification@nibss-plc.com.ng")
)

const (
	AlertBufferSize   = 100
	AlertThrottleTime = 5 * time.Minute
	MaxRetries        = 3
	FailureThresholdCount
	HTTPRequestTimeout = time.Duration(30) * time.Second
	ReportsDir         = "reports"
	ConsoleBaseURL     = "http://172.20.10.12:56865/console/"
)

var (
	HeaderBg    = []int{0, 32, 96}     // Deep Navy Blue
	TitleBg     = []int{0, 51, 153}    // Royal Blue
	TableBg     = []int{240, 244, 248} // Light Blue-Gray
	AlertColor  = []int{231, 76, 60}   // Red
	NormalColor = []int{46, 204, 113}  // Green
)

const (
	Healthy = iota
	Escalation
	Acknowledged
	Degraded
	UnknownStatus
	Scheduled
)

var StatusDescriptions = map[int]string{
	Healthy:      "Active Systems",
	Degraded:     "Inactive Systems",
	Acknowledged: "Inactive Acknowledged Systems",
	Scheduled:    "Scheduled for Maintenance",
}
