package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

// Define structures for different notification platform configurations

// NotificationPlatform represents the type of notification service
type NotificationPlatform string

const (
	Email    NotificationPlatform = "email"
	Slack    NotificationPlatform = "slack"
	SMS      NotificationPlatform = "sms"
	Discord  NotificationPlatform = "discord"
	Telegram NotificationPlatform = "telegram"
)

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
	TLSRequired bool   `json:"tls_required"`
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

// ConfigManager handles loading and accessing notification configurations
type ConfigManager struct {
	config NotificationConfig
}

func (cm *ConfigManager) LoadConfig(configPath string) error {
	// Ensure the file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist: %s", configPath)
	}

	// Read and parse the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("error reading config file: %w", err)
	}

	if err := json.Unmarshal(data, &cm.config); err != nil {
		return fmt.Errorf("error parsing config file: %w", err)
	}

	return cm.validate()
}

// validate checks if the loaded configuration is valid
func (cm *ConfigManager) validate() error {
	// Validate Email configuration if enabled
	if cm.config.Email.Enabled {
		if cm.config.Email.SMTPServer == "" || cm.config.Email.SMTPPort == 0 {
			return fmt.Errorf("invalid email configuration: SMTP server and port are required")
		}
	}

	// Validate Slack configuration if enabled
	if cm.config.Slack.Enabled {
		if cm.config.Slack.WebhookURL == "" && cm.config.Slack.BotToken == "" {
			return fmt.Errorf("invalid slack configuration: either webhook URL or bot token is required")
		}
	}

	// Validate SMS configuration if enabled
	if cm.config.SMS.Enabled {
		if cm.config.SMS.Provider == "" || cm.config.SMS.AccountSID == "" || cm.config.SMS.AuthToken == "" {
			return fmt.Errorf("invalid SMS configuration: provider, account SID, and auth token are required")
		}
	}

	return nil
}

// GetEmailConfig returns the email configuration
func (cm *ConfigManager) GetEmailConfig() EmailConfig {
	return *cm.config.Email
}

// GetSlackConfig returns the Slack configuration
func (cm *ConfigManager) GetSlackConfig() SlackConfig {
	return *cm.config.Slack
}

//
//func DeserializeConfig(configType string, configData string) (NotificationConfig, error) {
//	switch configType {
//	case "email":
//		var emailConfig NotificationConfig
//		err := json.Unmarshal([]byte(configData), &emailConfig.Email)
//		return emailConfig, err
//	case "slack":
//		var slackConfig NotificationConfig
//		err := json.Unmarshal([]byte(configData), &slackConfig.Slack)
//		return slackConfig, err
//	case "sms":
//		var smsConfig NotificationConfig
//		err := json.Unmarshal([]byte(configData), &smsConfig.SMS)
//		return smsConfig, err
//	default:
//		return nil, fmt.Errorf("unknown config type: %s", configType)
//	}
//}
