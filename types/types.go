package mstypes

import (
	"github.com/google/uuid"
	"github.com/slack-go/slack"
)

// AgentInfo represents the complete metrics information for an agent
type AgentInfo struct {
	Version   string `json:"version"`
	AgentID   string `json:"agent_id"`
	IPAddress string `json:"IPAddress"`
	Name      string `json:"name"`
	OS        string `json:"os"`
	//LastSync   sql.NullTime `json:"lastSync"`
	SDKVersion string `json:"SDKVersion"`
	Metrics    []Metric
	Disks      []DiskMetric
}

// AgentMetricResponse represents the complete metrics data for an agent
type AgentMetricResponse struct {
	Status     string     `json:"status"`
	SystemInfo SystemInfo `json:"systemInfo"`
	Uptime     string     `json:"uptime"`
	AgentInfo  AgentInfo  `json:"agent_info"`
}

type DiskMetric struct {
	Drive      string `json:"drive"`
	Size       int64  `json:"size"`
	Free       int64  `json:"free"`
	Used       int64  `json:"used"`
	FormatSize string `json:"formatSize"`
	FormatFree string `json:"formatFree"`
}

// SystemInfo represents system metrics (CPU, memory, disk)
type SystemInfo struct {
	CPU    [][]float64  `json:"cpu"`    // Each entry is [timestamp, usage]
	Memory [][]float64  `json:"memory"` // Each entry is [timestamp, usage]
	Disk   []DiskMetric `json:"disk"`
}

type Metric struct {
	Timestamp    int64
	TimestampMem int64
	CPUUsage     float64
	MemoryUsage  float64
	AgentID      string
}

type AgentRepositoryMetric struct {
	AgentHostAddress       string
	AgentHostName          string
	CpuUsage               float64
	MemoryUsage            float64
	CurrentDiskUtilization float64
	TotalStorageCapacity   string
	AgentAPI               string
}

type AgentEscalation struct {
	AppId   uuid.UUID `json:"appId"`
	AgentId string    `json:"agentId"`
	Metric  string    `json:"metric"`
	Cause   string    `json:"cause"`
}

// ServerResource represents the server resource utilization data
type ServerResource struct {
	ServerName        string
	CPUUtilization    float64
	MemoryUtilization float64
	DiskUtilization   float64
}

// NetworkDevice represents the network device bandwidth utilization data
type NetworkDeviceMetric struct {
	SystemMonitorId   string
	DeviceName        string
	DeviceIP          string
	MetricName        string
	MetricDescription string
	MetricValue       any
	LastPoll          string
	// Uptime               string
	// Interfaces           string
	// BandwidthUtilization float64
	// Description          string
	// CPUUsage             float64
	// MemoryUsage          float64
	// InboundTraffic       float64
	// OutboundTraffic      float64
}

// ProcessResourceUsage represents a single process entry returned by the Agent API endpoint.
type ProcessResourceUsage struct {
	Username      string  `json:"username"`
	PID           int     `json:"pid"`
	CPUPercent    float64 `json:"cpu_percent"`
	CreateTime    float64 `json:"create_time"`
	Status        string  `json:"status"`
	MemoryPercent float64 `json:"memory_percent"`
	Name          string  `json:"name"`
}

type ProcessResponse []ProcessResourceUsage

type NotificationRecipient struct {
	SystemMonitorId uuid.UUID `json:"systemMonitorId"`
	ServiceName     string    `json:"serviceName"`
	UserName        string
	Email           string
	PhoneNumber     string
	SlackId         string
	GroupName       string
	Platform        string
}

type NotificationRecipients struct {
	Users []NotificationRecipient
	//Groups      []Group
}

type User struct {
	Email string
	Name  string
}

type Metrics struct {
	CPU     bool `json:"CPU"`
	Disk    bool `json:"Disk"`
	Memory  bool `json:"Memory"`
	Latency bool `json:"Latency"`
}

type MetricsThreshold struct {
	CPU struct {
		High int `json:"high"`
		Low  int `json:"low"`
		Mid  int `json:"mid"`
	} `json:"CPU"`
	Disk struct {
		High int `json:"high"`
		Low  int `json:"low"`
		Mid  int `json:"mid"`
	} `json:"Disk"`
	Latency struct {
		High int `json:"high"`
		Low  int `json:"low"`
		Mid  int `json:"mid"`
	} `json:"Latency"`
}

type Config struct {
	AgentID            string           `json:"AgentID"`
	AgentSleepInterval int              `json:"AgentSleepInterval"`
	ProvisionedState   string           `json:"ProvisionedState"`
	LicenseKey         string           `json:"LicenseKey"`
	APIBaseUrl         string           `json:"APIBaseUrl"`
	Metrics            Metrics          `json:"Metrics"`
	MetricsThreshold   MetricsThreshold `json:"MetricsThreshold"`
	AgentMonitoredLogs []interface{}    `json:"AgentMonitoredLogs"` // Use interface{} to handle null or any type
}

type AgentThresholdResponse struct {
	Config Config `json:"config"`
}

type AgentContainerResponse struct {
}

// import (
// 	"database/sql"
// 	"log"

// 	"github.com/ZEGIFTED/MS.GoMonitor/internal"
// 	"github.com/slack-go/slack"
// )

// NotificationPlatform represents the type of notification service
type NotificationPlatform string

const (
	Email    NotificationPlatform = "email"
	Slack    NotificationPlatform = "slack"
	SMS      NotificationPlatform = "sms"
	Discord  NotificationPlatform = "discord"
	Telegram NotificationPlatform = "telegram"
)

// Define structures for different notification platform configurations

// BaseConfig contains common configuration fields
type BaseConfig struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`
}

// EmailConfig contains email-specific configuration
type EmailConfig struct {
	BaseConfig
	SMTPServer  string `json:"smtp_server"`
	SMTPPort    int    `json:"smtp_port"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	FromAddress string `json:"from_address"`
	UseTLS      bool   `json:"use_tls"`
}

type UserData struct {
	Name           string
	RecipientGroup string
}

type Link struct {
	Text   string
	URL    string
	NewTab bool
}

type Logo struct {
	UseSVG   bool
	ImageURL string
	// Width          string
	// Height         string
	// FooterWidth    string
	// FooterHeight   string
	Text           string
	PrimaryColor   string
	SecondaryColor string
}

type MetaData struct {
	Year         int
	CompanyName  string
	Timestamp    string
	FooterLinks  []Link
	SupportEmail string
	SupportPhone string
}

type EmailTemplateData struct {
	Title   string
	Heading string
	Content string

	ServiceName string

	User                UserData
	ActionURL           string
	DashboardMonitorURL string

	Items            []string
	TableData        [][]string
	ProcessTableData ProcessResponse
	ExtraFields      map[string]interface{}

	Logo Logo
	Meta MetaData
}

type SlackMessage struct {
	Blocks      []slack.Block      `json:"blocks"`
	Text        string             `json:"text"`
	Attachments []slack.Attachment `json:"attachments"`
}

// SlackConfig contains Slack-specific configuration
type SlackConfig struct {
	BaseConfig
	WebhookURL  string   `json:"webhook_url"`
	BotToken    string   `json:"bot_token"`
	Channels    []string `json:"channels"`
	DefaultUser string   `json:"default_user"`
}

// SMSConfig contains SMS-specific configuration
type SMSConfig struct {
	BaseConfig
	Provider      string `json:"provider"`
	AccountSID    string `json:"account_sid"`
	AuthToken     string `json:"auth_token"`
	FromNumber    string `json:"from_number"`
	MessagePrefix string `json:"message_prefix"`
}

// NotificationConfig holds all platform configurations
type NotificationConfig struct {
	Email *EmailConfig `json:"email,omitempty"`
	Slack *SlackConfig `json:"slack,omitempty"`
	SMS   *SMSConfig   `json:"sms,omitempty"`
	//OtherConfigs map[string]interface{} `json:"other_configs,omitempty"` // For any other configs
}

// TeamsMessage represents the message structure for Teams webhook
type TeamsMessage struct {
	Type       string `json:"@type"`
	Context    string `json:"@context"`
	Summary    string `json:"summary,omitempty"`
	ThemeColor string `json:"themeColor,omitempty"`
	Title      string `json:"title"`
	Text       string `json:"text"`
}
