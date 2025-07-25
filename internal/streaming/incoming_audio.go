package streaming

import (
	"encoding/json"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/gorilla/websocket"
)

type TwilioMessage struct {
	Event           string `json:"event,omitempty"`
	SequenceNumber  string `json:"sequenceNumber,omitempty"`
	Protocol        string `json:"protocol,omitempty"`
	Version         string `json:"version,omitempty"`
	Start           *StartMessage `json:"start,omitempty"`
	Media           *MediaMessage `json:"media,omitempty"`
	Mark            *MarkMessage `json:"mark,omitempty"`
	StreamSid       string `json:"streamSid,omitempty"`
}

type MarkMessage struct {
	Name string `json:"name"`
}

type StartMessage struct {
	AccountSid      string   `json:"accountSid,omitempty"`
	StreamSid       string   `json:"streamSid,omitempty"`
	CallSid         string   `json:"callSid,omitempty"`
	Tracks          []string `json:"tracks,omitempty"`
	MediaFormat     MediaFormat `json:"mediaFormat,omitempty"`
	CustomParameters map[string]interface{} `json:"customParameters,omitempty"`
}

type MediaFormat struct {
	Encoding   string `json:"encoding,omitempty"`
	SampleRate int    `json:"sampleRate,omitempty"`
	Channels   int    `json:"channels,omitempty"`
}

type MediaMessage struct {
	Track     string `json:"track,omitempty"`
	Chunk     string `json:"chunk,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Payload   string `json:"payload,omitempty"`
}

func (c *CallOrchestrator) handleInboundAudio() {
	for {
		messageType, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				logger.S.Info("WebSocket connection closed")
			} else {
				logger.S.Error("Error reading WebSocket message:", err)
			}
			if c.call.DisconnectReason == "" {
				c.call.DisconnectReason = "user_hangup"
			}
			c.doneChan <- true
			break
		}

		if messageType == websocket.TextMessage {
			var twilioMessage TwilioMessage
			err := json.Unmarshal(message, &twilioMessage)
			if err != nil {
				logger.S.Error("Error unmarshalling Twilio message:", err)
				continue
			}

			if twilioMessage.Start != nil {
				c.startChan <- twilioMessage
			}

			// If the message is a media message, send the payload to the audioData channel
			if twilioMessage.Media != nil {
				c.audioChan <- twilioMessage.Media.Payload
				c.rtcAudioChan <- twilioMessage.Media.Payload
			}

			if twilioMessage.Mark != nil {
				delete(c.marks, twilioMessage.Mark.Name)
				// If we have no more marks - then it's the user's turn to talk
				if len(c.marks) == 0 {
					c.turn = "user"
				}
			}
		} else if messageType == websocket.BinaryMessage {
			// Log binary messages from Twilio
			logger.S.Info("Received binary message")
		}
	}
}
