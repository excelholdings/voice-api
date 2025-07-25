package webhook

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/flyflow-devs/flyflow/internal/models"
)

type Event struct {
	EventName string       `json:"event"`
	Call      *models.Call `json:"call"`
	Data      interface{}  `json:"data,omitempty"`
}

func EmitEvent(webhookurl string, name string, call *models.Call, data interface{}) {
	go func() {
		// Validate the URL
		if _, err := url.ParseRequestURI(webhookurl); err != nil {
			return
		}

		// Create the event payload
		event := Event{
			EventName: name,
			Call:      call,
			Data:      data,
		}

		// Marshal the event payload to JSON
		payload, err := json.Marshal(event)
		if err != nil {
			logger.S.Errorf("Failed to marshal event payload: %s", err)
			return
		}

		// Create a new HTTP request
		req, err := http.NewRequest(http.MethodPost, webhookurl, bytes.NewBuffer(payload))
		if err != nil {
			logger.S.Errorf("Failed to create HTTP request: %s", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		// Create a backoff configuration
		backoffConfig := backoff.NewExponentialBackOff()
		backoffConfig.MaxElapsedTime = 1 * time.Minute

		// Create a backoff client
		client := &http.Client{}

		// Send the event with backoff retries
		err = backoff.Retry(func() error {
			resp, err := client.Do(req)
			if err != nil {
				logger.S.Errorf("Failed to send webhook event: %s", err)
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}

			logger.S.Errorf("Webhook event failed with status code: %d", resp.StatusCode)
			return backoff.Permanent(err)
		}, backoffConfig)

		if err != nil {
			logger.S.Errorf("Webhook event failed after retries: %s", err)
		}
	}()
}