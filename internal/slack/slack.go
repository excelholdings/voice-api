package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const webhookURL = "<placeholder>"

type SlackPayload struct {
	Text string `json:"text"`
}

func PostMessage(message string) error {
	payload := SlackPayload{
		Text: message,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("error sending POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	fmt.Println("Message posted to Slack successfully")
	return nil
}