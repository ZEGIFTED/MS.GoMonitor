package messaging

import (
	"database/sql"
	"log"

	"github.com/slack-go/slack"
)

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

	Items       []string
	TableData   [][]string
	ExtraFields map[string]interface{}

	Logo Logo
	Meta MetaData
}

// SlackClient handles Slack API interactions
type SlackClient struct {
	client *slack.Client
	logger *log.Logger
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

// NotificationManager handles loading and accessing notification configurations
type NotificationManager struct {
	Config *NotificationConfig
	Logger *log.Logger
	DB     *sql.DB
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
