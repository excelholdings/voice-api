package streaming

import (
	"context"
	"encoding/base64"
	api "github.com/deepgram/deepgram-go-sdk/pkg/api/live/v1/interfaces"
	"github.com/deepgram/deepgram-go-sdk/pkg/client/interfaces"
	client "github.com/deepgram/deepgram-go-sdk/pkg/client/live"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"strconv"
	"time"
)

func (c *CallOrchestrator) Message(mr *api.MessageResponse) error {
	if mr.Channel.Alternatives[0].Confidence > 0 {
		logger.S.Infof("transcript: %v", mr.Channel.Alternatives[0].Transcript)
	}
	// Don't send transcripts while we're generating a response
	if mr.Channel.Alternatives[0].Transcript != "" && mr.IsFinal == true {
		c.userSpeaking = false
		c.interruptionChan <- true
		c.metrics.startProcessing()
		c.userLastSpoke = time.Now()
		c.lastFinalizedMessage = time.Now()
		if c.turn != "assistant" {
			c.transcriptionsChan <- mr.Channel.Alternatives[0].Transcript
		}
	}

	if mr.Channel.Alternatives[0].Transcript != "" && mr.IsFinal == false && mr.Channel.Alternatives[0].Confidence > 0.5 {
		// Sometimes deepgram sends an unfinalized and then finalized transcript in rapid order,
		if time.Since(c.lastFinalizedMessage) > 2 * time.Second {
			c.userSpeaking = true
			c.interruptionChan <- true
			c.userLastSpoke = time.Now()
			c.turn = "user"
		}

	}

	return nil
}

func (c *CallOrchestrator) Open(ocr *api.OpenResponse) error {
	return nil
}

func (c *CallOrchestrator) Metadata(md *api.MetadataResponse) error {
	return nil
}

func (c *CallOrchestrator) SpeechStarted(ssr *api.SpeechStartedResponse) error {
	return nil
}

func (c *CallOrchestrator) UtteranceEnd(ur *api.UtteranceEndResponse) error {
	return nil
}

func (c *CallOrchestrator) Close(ocr *api.CloseResponse) error {
	return nil
}

func (c *CallOrchestrator) Error(er *api.ErrorResponse) error {
	// handle the error
	logger.S.Errorf("error from deepgram: %v", er.Message)
	return nil
}

func (c *CallOrchestrator) UnhandledEvent(byData []byte) error {
	return nil
}

func (c *CallOrchestrator) handleTranscripts() {
	ctx := context.Background()

	// client options
	cOptions := interfaces.ClientOptions{
		EnableKeepAlive: true,
		ApiKey: c.cfg.DeepgramAPIKey,
	}

	var endpointing uint
	if c.agent.Endpointing == 0 {
		endpointing = 100
	} else {
		endpointing = c.agent.Endpointing
	}

	strEndpointing := strconv.FormatUint(uint64(endpointing), 10)

	language := "en-US"
	if c.agent.Language != "" {
		language = c.agent.Language
	}

	// set the Transcription options
	tOptions := interfaces.LiveTranscriptionOptions{
		Model:       "nova-2",
		Language:    language,
		Punctuate:   true,
		Encoding:    "mulaw",
		Channels:    1,
		SampleRate:  8000,
		//Multichannel: true,
		//SmartFormat: true,
		InterimResults: true,
		//UtteranceEndMs: "2000",
		VadEvents:      true,
		Endpointing:    strEndpointing,
	}



	// create a Deepgram client
	dgClient, err := client.New(ctx, "", cOptions, tOptions, c)
	if err != nil {
		logger.S.Errorf("error creating deepgram client: %v", err)
		c.doneChan <- true
		return
	}

	// connect the websocket to Deepgram
	wsconn := dgClient.Connect()
	if wsconn == nil {
		logger.S.Errorf("deepgram client connection failed")
		c.doneChan <- true
		return
	}

	for {
		if c.done {
			break
		}

		data, _ := <-c.audioChan

		chunk, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			logger.S.Error("Error decoding base64 payload:", err)
			continue
		}

		if _, err := dgClient.Write(chunk); err != nil {
			logger.S.Error("error writing to the deegram client", err)
			continue
		}
	}
}