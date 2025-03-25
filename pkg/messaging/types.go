package messaging

import (
	"log"

	"github.com/slack-go/slack"
)

// SlackClient handles Slack API interactions
type SlackClient struct {
	client *slack.Client
	logger *log.Logger
}
