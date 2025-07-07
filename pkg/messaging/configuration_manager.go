package messaging

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mstypes "github.com/ZEGIFTED/MS.GoMonitor/types"
)

// NotificationManager handles loading and accessing notification configurations
type NotificationManager struct {
	Config *mstypes.NotificationConfig
	Logger *log.Logger
	DB     *sql.DB
}

func (cfgManager *NotificationManager) LoadConfig() error {
	log.Println("Loading Notification Manager configuration")

	if cfgManager.DB == nil {
		return fmt.Errorf("configuration Database is not initialized")
	}

	ctx := context.Background()

	query := `
	SELECT "Name",
      "Configuration"::json AS "Configuration"
	FROM "NotificationPlatforms";
`

	rows, err := cfgManager.DB.QueryContext(ctx, query)

	if err != nil {
		fmt.Println(err.Error())

		return err
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			cfgManager.Logger.Fatalf("DB Closure failed: %s", err.Error())
		}
	}(rows)

	//var config NotificationHandler

	for rows.Next() {
		var ConfigName string
		var Configuration string

		err := rows.Scan(&ConfigName, &Configuration)

		if err != nil {
			cfgManager.Logger.Printf("⚠️ Load Notification Config failed: %v", err)
		}

		if strings.Contains(ConfigName, "Email") {
			if err := json.Unmarshal([]byte(Configuration), &cfgManager.Config.Email); err != nil {
				//return NotificationHandler{}, fmt.Errorf("error parsing email config: %w", err)
				cfgManager.Logger.Printf("Warning: error parsing email config %s: %v", err.Error(), Configuration)
				continue
			}
		}

		if strings.Contains(ConfigName, "Slack") {
			if err := json.Unmarshal([]byte(Configuration), &cfgManager.Config.Slack); err != nil {
				//return NotificationHandler{}, fmt.Errorf("error parsing email config: %w", err)
				cfgManager.Logger.Printf("Warning: error parsing slack config %s: %v", err.Error(), Configuration)
				continue
			}
		}
	}

	//config.Slack = cfgManager.config.Slack

	//err_ := cfgManager.validate()
	//if err_ != nil {
	//	return err_
	//}

	return nil
}

// validate checks if the loaded configuration is valid
func (cfgManager *NotificationManager) Validate() error {
	if cfgManager.Config.Email == nil && cfgManager.Config.Slack == nil {
		return fmt.Errorf("❌ Notification config validation failed")
	}

	// Validate Email configuration if enabled
	if cfgManager.Config.Email.Enabled {
		if cfgManager.Config.Email.SMTPServer == "" || cfgManager.Config.Email.SMTPPort == 0 {
			cfgManager.Logger.Panicf("invalid email configuration: SMTP server and port are required")
		}
	}

	// Validate Slack configuration if enabled
	if cfgManager.Config.Slack.Enabled {
		if cfgManager.Config.Slack.WebhookURL == "" && cfgManager.Config.Slack.BotToken == "" {
			cfgManager.Logger.Panicf("invalid Slack configuration: either webhook URL or bot token is required")
		}
	}

	//// Validate SMS configuration if enabled
	//if cm.config.SMS.Enabled {
	//	if cm.config.SMS.Provider == "" || cm.config.SMS.AccountSID == "" || cm.config.SMS.AuthToken == "" {
	//		return fmt.Errorf("invalid SMS configuration: provider, account SID, and auth token are required")
	//	}
	//}

	return nil
}

// GetEmailConfig returns the email configuration
func (cfgManager *NotificationManager) GetEmailConfig() mstypes.EmailConfig {
	return *cfgManager.Config.Email
}

// GetSlackConfig returns the Slack configuration
func (cfgManager *NotificationManager) GetSlackConfig() mstypes.SlackConfig {
	return *cfgManager.Config.Slack
}

//
//func DeserializeConfig(configType string, configData string) (NotificationHandler, error) {
//	switch configType {
//	case "email":
//		var emailConfig NotificationHandler
//		err := json.Unmarshal([]byte(configData), &emailConfig.Email)
//		return emailConfig, err
//	case "slack":
//		var slackConfig NotificationHandler
//		err := json.Unmarshal([]byte(configData), &slackConfig.Slack)
//		return slackConfig, err
//	case "sms":
//		var smsConfig NotificationHandler
//		err := json.Unmarshal([]byte(configData), &smsConfig.SMS)
//		return smsConfig, err
//	default:
//		return nil, fmt.Errorf("unknown config type: %s", configType)
//	}
//}
