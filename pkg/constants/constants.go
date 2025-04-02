package constants

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	// "github.com/joho/godotenv"
)

func init() {
	// Load .env file
	// err := godotenv.Load(".env")

	// if err != nil {
	// 	log.Fatalf("Warning: Error loading Env file. %s", err.Error())
	// }

	// Ensure all env vars are uppercase
	// for _, e := range os.Environ() {
	// 	pair := strings.SplitN(e, "=", 2)
	// 	if len(pair) == 2 {
	// 		upperKey := strings.ToUpper(pair[0])
	// 		if upperKey != pair[0] {
	// 			os.Setenv(upperKey, pair[1])
	// 		}
	// 	}
	// }
	// log.Println("Environment Variables Loaded:", os.Environ())
}

// GetEnvWithDefault retrieves an environment variable with a fallback default value
func GetEnvWithDefault(key, defaultValue string) string {
	// Try exact case first
	if value := os.Getenv(key); value != "" {
		log.Printf("Found env %s=%s", key, value)
		return value
	}

	// Try uppercase version if different
	upperKey := strings.ToUpper(key)
	if upperKey != key {
		if value, exists := os.LookupEnv(upperKey); exists {
			log.Printf("Found env %s=%s (via uppercase conversion)", upperKey, value)
			return value
		}
	}

	// Try lowercase version if different
	lowerKey := strings.ToLower(key)
	if lowerKey != key {
		if value, exists := os.LookupEnv(lowerKey); exists {
			log.Printf("Found env %s=%s (via lowercase conversion)", lowerKey, value)
			return value
		}
	}

	log.Printf("Env key '%s' not found, using default: '%s'", key, defaultValue)
	return defaultValue
}

type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

var DB = DBConfig{
	Host:     GetEnvWithDefault("DBHOST", "localhost"),
	Port:     GetEnvWithDefault("DB_PORT", "1433"),
	Name:     GetEnvWithDefault("DB_NAME", "MS"),
	User:     GetEnvWithDefault("DB_USER", "sa"),
	Password: GetEnvWithDefault("DB_PASSWORD", "dbuser123$"),
}

var (
	LogPath            = "logs/"
	LogFileName        = LogPath + "ms-svc_monitor.log"
	DatabaseConnString = fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s;", DB.Host, DB.User, DB.Password, DB.Port, DB.Name)
	SMTPHost           = GetEnvWithDefault("MAILHOST", "localhostgd")
	SMTPPort           = GetEnvWithDefault("MAILPORT", "2532")
	SMTPUser           = GetEnvWithDefault("MAILUSER", "test-notification@nibss-plc.com.ng")
	SMTPPass           = GetEnvWithDefault("MAILPASS", "password123$_")
	STMP_ADMIN_MAIL    = GetEnvWithDefault("STMPADMIN", "test-notification@nibss-plc.com.ng2")
)

const (
	AlertBufferSize   = 100
	AlertThrottleTime = 5 * time.Minute
	MaxRetries        = 3
	FailureThresholdCount
	HTTPRequestTimeout = time.Duration(30) * time.Second
	ReportsDir         = "reports"
	ConsoleBaseURL     = "http://172.20.10.12:56865/console/"
	OrganizationName   = "NIBSS MS"
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
