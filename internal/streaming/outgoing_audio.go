package streaming

import (
	"encoding/base64"
	"github.com/flyflow-devs/flyflow/internal/clients"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/flyflow-devs/flyflow/internal/voices"
	"github.com/google/uuid"
	"io"
	"time"

	"github.com/haguro/elevenlabs-go"
)

func (c *CallOrchestrator) handleOutgoingAudio() {
	elevenlabs.SetAPIKey(c.cfg.ElevenLabsAPIKey)
	elevenlabs.SetTimeout(1 * time.Minute)

	cartesiaClient := clients.NewCartesiaClient(c.cfg)

	if c.agent.VoiceOptimization == 0 {
		c.agent.VoiceOptimization = 3
	}

	for {
		if c.done {
			break
		}
		response, _ := <-c.responseChan

		if isElevenLabsVoice(c.agent.VoiceId) {
			c.streamElevenLabsAudio(response)
		} else if isCartesiaVoice(c.agent.VoiceId) {
			c.streamCartesiaAudio(cartesiaClient, response)
		} else {
			logger.S.Errorf("Unknown voice service for voice ID: %s", c.agent.VoiceId)
		}
	}
}

func (c *CallOrchestrator) streamElevenLabsAudio(response string) {
	model := "eleven_turbo_v2_5"
	if c.agent.Language != "en-US" && c.agent.Language != "" {
		model = "eleven_turbo_v2_5"
	}

	if err := elevenlabs.TextToSpeechStream(
		c,
		voices.Voices[c.agent.VoiceId],
		elevenlabs.TextToSpeechRequest{
			Text:    response,
			ModelID: model,
		},
		elevenlabs.OutputFormat("ulaw_8000"),
		elevenlabs.LatencyOptimizations(int(c.agent.VoiceOptimization))); err != nil {
		logger.S.Errorf("error streaming speech from eleven labs: %v", err)
	}
}

func (c *CallOrchestrator) streamCartesiaAudio(client *clients.CartesiaClient, response string) {
	modelID := "sonic-english"
	language := "en"

	if c.agent.Language != "en-US" && c.agent.Language != "" {
		modelID = "sonic-multilingual"
		language = mapLanguageCode(c.agent.Language)
	}

	stream, err := client.StreamSpeechBytes(modelID, response, voices.CartesiaVoices[c.agent.VoiceId], language)
	if err != nil {
		logger.S.Errorf("error streaming speech from Cartesia: %v", err)
		return
	}
	defer stream.Close()

	buffer := make([]byte, 4096)
	for {
		n, err := stream.Read(buffer)
		if err == io.EOF {
			c.writeToTwilio(buffer[:n])
			break
		}
		if err != nil {
			logger.S.Errorf("Error reading Cartesia audio stream: %v", err)
			break
		}

		c.writeToTwilio(buffer[:n])
	}
}

func (c *CallOrchestrator) writeToTwilio(p []byte) {
	if c.userSpeaking {
		return
	}

	c.metrics.stopProcessing()
	encodedMessage := base64.StdEncoding.EncodeToString(p)

	message := TwilioMessage{
		Event:     "media",
		StreamSid: c.streamSid,
		Media: &MediaMessage{
			Payload: encodedMessage,
		},
	}

	markUUID, _ := uuid.NewUUID()
	markUUIDString := markUUID.String()
	mark := TwilioMessage{
		Event: "mark",
		StreamSid: c.streamSid,
		Mark: &MarkMessage{
			Name: markUUIDString,
		},
	}

	c.outgoingWebsocketLock.Lock()
	defer c.outgoingWebsocketLock.Unlock()

	if err := c.conn.WriteJSON(message); err != nil {
		logger.S.Errorf("Error writing Twilio message: %v", err)
	}
	c.marks[markUUIDString] = struct{}{}
	if err := c.conn.WriteJSON(mark); err != nil {
		logger.S.Errorf("Error writing Twilio message: %v", err)
	}
}

func (c *CallOrchestrator) Write(p []byte) (n int, err error) {
	c.writeToTwilio(p)
	return len(p), nil
}

func isElevenLabsVoice(voiceID string) bool {
	_, ok := voices.Voices[voiceID]
	return ok
}

func isCartesiaVoice(voiceID string) bool {
	_, ok := voices.CartesiaVoices[voiceID]
	return ok
}

func mapLanguageCode(languageCode string) string {
	switch languageCode {
	case "es-ES":
		return "es"
	case "fr-FR":
		return "fr"
	case "de-DE":
		return "de"
	case "pt-BR":
		return "pt"
	case "zh-CN":
		return "zh"
	case "ja-JP":
		return "ja"
	default:
		return "en"
	}
}