package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ZEGIFTED/MS.GoMonitor/pkg/messaging"
	"net/http"
)

// MessageType defines the type of message being sent
type MessageType string

const (
	Individual MessageType = "individual"
	Channel    MessageType = "channel"
)

// MessageGroup represents a group of messages to be sent
type MessageGroup struct {
	WebhookURL string
	Type       MessageType
	Recipients []string
	Message    messaging.TeamsMessage
}

// SendTeamsMessage sends a message to a Microsoft Teams channel or group using a webhook URL
func SendTeamsMessage(webhookURL string, message messaging.TeamsMessage) error {

	// Construct the message body with mentions
	//var formattedMessage string
	//if len(mentionList) > 0 {
	//	for _, mention := range mentionList {
	//		formattedMessage += fmt.Sprintf("<at>%s</at> ", mention)
	//	}
	//	formattedMessage += message
	//} else {
	//	formattedMessage = message
	//}

	// Convert the message to JSON
	messageJSON, err := json.Marshal(message)

	if err != nil {
		return fmt.Errorf("error marshaling message: %v", err)
	}

	// Send the HTTP POST request to the webhook URL
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(messageJSON))
	if err != nil {
		return fmt.Errorf("error sending message to Teams: %v", err)
	}
	defer resp.Body.Close()

	// Check the response status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
	}

	return nil
}
