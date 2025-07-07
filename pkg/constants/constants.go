package constants

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

// GetEnvWithDefault retrieves an environment variable with a fallback default value
func GetEnvWithDefault(key, defaultValue string) string {
	// Try exact case first
	if value := os.Getenv(key); value != "" {
		// log.Printf("Found env %s=%s", key, value)
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
	Host:     GetEnvWithDefault("DBHOST", ""),
	Port:     GetEnvWithDefault("DB_PORT", ""),
	Name:     GetEnvWithDefault("DB_NAME", ""),
	User:     GetEnvWithDefault("DB_USER", ""),
	Password: GetEnvWithDefault("DB_PASSWORD", ""),
}

var (
	PluginDirs = []string{
		"./plugins", // Local development
		// "/etc/ms-monitor/plugins", // System-wide installation
		// "/app/plugins",            // Docker container default
	}
	MaxPluginsPerService = 3
	LogPath              = "logs/"
	LogFileName          = LogPath + "ms-svc_monitor.log"
	DatabaseConnString   = fmt.Sprintf(
		"host=%s user=%s password=%s port=%s dbname=%s sslmode=disable",
		DB.Host, DB.User, DB.Password, DB.Port, DB.Name,
	)
	SMTPHost        = GetEnvWithDefault("MAILHOST", "localhost")
	SMTPPort        = GetEnvWithDefault("MAILPORT", "25")
	SMTPUser        = GetEnvWithDefault("MAILUSER", "")
	SMTPPass        = GetEnvWithDefault("MAILPASS", "$_")
	STMP_ADMIN_MAIL = GetEnvWithDefault("STMPADMIN", "")
)

const (
	DefaultCronExpression = "*/15 * * * *"
	AlertBufferSize       = 100
	AlertThrottleTime     = 5 * time.Minute
	MaxRetries            = 3
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

type StatusInfo struct {
	Name        string
	Description string
	Color       string
	Flag        int
}

var StatusMap = map[int]StatusInfo{
	UnknownStatus: {
		Name:        "Unknown",
		Description: "Unknown System Status",
		Color:       "#6b7280", // Gray
		Flag:        UnknownStatus,
	},
	Healthy: {
		Name:        "Healthy",
		Description: "Active System",
		Color:       "#10B981", // Green
		Flag:        Healthy,
	},
	Escalation: {
		Name:        "Escalation",
		Description: "System Requires Attention",
		Color:       "#f59e0b", // Amber
		Flag:        Escalation,
	},
	Acknowledged: {
		Name:        "Acknowledged",
		Description: "Acknowledged Inactive System",
		Color:       "#3b82f6", // Blue
		Flag:        Acknowledged,
	},
	Degraded: {
		Name:        "Degraded",
		Description: "Inactive Systems",
		Color:       "#ef4444", // Red
		Flag:        Degraded,
	},
	InvalidConfiguration: {
		Name:        "Unknown",
		Description: "Invalid Service Configuration",
		Color:       "#6b7280", // Gray
		Flag:        InvalidConfiguration,
	},
	Scheduled: {
		Name:        "Scheduled",
		Description: "Scheduled for Maintenance",
		Color:       "#8b5cf6", // Purple
		Flag:        Scheduled,
	},
}

func GetStatusInfo(code int, addedMessage string) StatusInfo {
	if info, ok := StatusMap[code]; ok {
		if addedMessage != "" {
			info.Description += " | " + addedMessage
		}
		return info
	}

	return StatusMap[UnknownStatus]
}

const (
	UnknownStatus = iota
	Healthy
	Escalation
	Acknowledged
	Degraded
	InvalidConfiguration
	Scheduled
)

var StatusDescriptions = map[int]string{
	Healthy:      "Active Systems",
	Degraded:     "Inactive Systems",
	Acknowledged: "Inactive Acknowledged Systems",
	Scheduled:    "Scheduled for Maintenance",
}

type ServiceType string

const (
	HTTPService           ServiceType = "http"
	DatabaseService       ServiceType = "database"
	ServiceTypeRedis      ServiceType = "redis"
	ServiceTypeKafka      ServiceType = "kafka"
	ServiceTypeElastic    ServiceType = "elasticsearch"
	ServiceTypeKubernetes ServiceType = "kubernetes"
)
