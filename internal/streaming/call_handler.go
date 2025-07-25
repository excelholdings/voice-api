package streaming

import (
	"context"
	"encoding/json"
	"github.com/flyflow-devs/flyflow/internal/classifier"
	"github.com/flyflow-devs/flyflow/internal/config"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/flyflow-devs/flyflow/internal/models"
	"github.com/gorilla/websocket"
	"github.com/sashabaranov/go-openai"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
	"gorm.io/gorm"
	"sync"
	"time"
)

type CallOrchestrator struct {
	cfg *config.Config
	db *gorm.DB
	conn *websocket.Conn

	// Done signals that the call is over
	done     bool
	doneChan chan bool

	// Database objects
	agent *models.Agent
	call  *models.Call

	// Channels for streaming call data
	audioChan          chan string
	rtcAudioChan       chan string
	transcriptionsChan chan string
	interruptionChan   chan bool
	userSpeaking       bool
	responseChan       chan string
	startChan          chan TwilioMessage
	generatingText     bool

	// Metadata
	streamSid     string
	callSid       string
	metrics       *Metrics
	userLastSpoke time.Time
	lastFinalizedMessage time.Time

	classifier *classifier.Classifier

	marks map[string]interface{}
	outgoingWebsocketLock sync.Mutex

	// Who's turn is it to speak
	turn string
}

func NewCallOrchestrator(cfg *config.Config, db *gorm.DB, conn *websocket.Conn, classifier *classifier.Classifier) *CallOrchestrator {

	return &CallOrchestrator{
		cfg: cfg,
		db: db,
		conn: conn,

		doneChan: make(chan bool),
		done: false,

		audioChan:          make(chan string),
		rtcAudioChan:       make(chan string),
		transcriptionsChan: make(chan string),
		interruptionChan:   make(chan bool),
		responseChan:       make(chan string),
		startChan:          make(chan TwilioMessage),
		userSpeaking:       false,
		generatingText:     false,

		metrics: NewMetrics(),

		userLastSpoke: time.Now(),
		lastFinalizedMessage: time.Now(),

		classifier: classifier,

		marks: make(map[string]interface{}),

		outgoingWebsocketLock: sync.Mutex{},

		turn: "user",
	}
}

func (c *CallOrchestrator) OrchestrateCall() {
	go c.handleInboundAudio()

	startMessage := <- c.startChan
	c.streamSid = startMessage.StreamSid
	c.callSid = startMessage.Start.CallSid

	call, err := c.fetchCall()
	if err != nil {
		logger.S.Errorf("error fetching call, fatal to call, exiting: %v", err)
		return
	}

	if err := c.setAgent(call); err != nil {
		logger.S.Errorf("failed to set agent: %v", err)
	}
	if err := c.upsertCall(call); err != nil {
		logger.S.Errorf("failed to set agent: %v", err)
	}
	c.startCall()


	// Async handle the parts of the conversation
	go c.handleLLM()
	go c.handleTranscripts()
	go c.handleOutgoingAudio()
	go c.handleInterruption()
	go c.handleToolCalls()
	go c.handleUpdatingContext()
	go c.handleActions()
	go c.handleWebRTC()
	//go c.handleReminders()

	// Have the agent speak the initial message to the user
	if c.agent.InitialMessage != "" && !c.call.UserSpeaksFirst {
		c.metrics.startProcessing()
		c.responseChan <- c.agent.InitialMessage
	}

	// Wait for the agent to finish
	<-c.doneChan
	c.done = true

	// Make sure that the call is saved at the end
	if err := c.endCall(); err != nil {
		logger.S.Errorf("failed to end call: %v", err)
	}
}

func (c *CallOrchestrator) startCall() {
	c.call.StartedAt = time.Now()

	// Start recording the calls
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: c.cfg.TwilioAccountSid,
		Password: c.cfg.TwilioAccountAuthToken,
	})

	// Start recording the call
	bothString := "both"
	recording, err := client.Api.CreateCallRecording(c.callSid, &twilioApi.CreateCallRecordingParams{
		PathAccountSid: &c.cfg.TwilioAccountSid,
		RecordingTrack: &bothString,
	})
	if err != nil {
		logger.S.Errorf("error creating recording: %v", err)
	} else {
		c.call.RecordingSid = *recording.Sid
	}

	c.call.InProgress = true

	_ = c.saveCall()

	// Emit the call started callback
	c.EmitEvent("call_started", nil)
}

func (c *CallOrchestrator) endCall() error {
	c.call.EndedAt = time.Now()
	c.call.TimeSeconds = time.Since(c.call.StartedAt).Seconds()
	c.call.AverageLatency = c.metrics.getAverageLatency()

	c.calculateSentiment()

	c.call.InProgress = false

	if err := c.saveCall(); err != nil {
		return err
	}

	c.EmitEvent("call_ended", nil)

	return nil
}

func (c *CallOrchestrator) calculateSentiment() {
	openaiConfig := openai.DefaultConfig(c.cfg.OpenAIAPIKey)
	openaiClient := openai.NewClientWithConfig(openaiConfig)

	type response struct {
		Sentiment uint `json:"sentiment"`
	}

	transcript, _ := json.Marshal(c.call.Transcript)

	ctx := context.Background()
	resp, err := openaiClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    "gpt-4o",
			Messages: []openai.ChatCompletionMessage{
				{
					Role: "system",
					Content: `
						You are an expert at scoring sentiment from conversations. 

						INSTRUCTIONS 
						- Score the sentiment of the conversation 1-10
						- Return json and ONLY json (no markup etc) in the format {"sentiment": <score uint 1-10>}
						- Bias your scores towards a positive sentiment and only score negative if the transcript is truly negative. Even transcripts that are not explicitly positive should be scored as positive 

						SENTIMENT SCORES
						1-3 Negative 
						3-7 Neutral
						7-10 Positive
					`,
				},
				{
					Role: "user",
					Content: string(transcript),
				},
			},
		},
	)

	if err != nil {
		logger.S.Errorf("error computing sentiment from openai: %v", err)
	}

	sentiment := response{}

	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &sentiment); err != nil {
		logger.S.Errorf("error unmarshalling sentiment score, content: %v, error: %v", resp.Choices[0].Message.Content, err)
	}

	c.call.Sentiment = sentiment.Sentiment
}

func (c *CallOrchestrator) saveCall() error {
	return c.db.Save(c.call).Error
}

func (c *CallOrchestrator) upsertCall(twilioCall *twilioApi.ApiV2010Call) error {
	toPhoneNumber := *twilioCall.To
	fromPhoneNumber := *twilioCall.From

	var call models.Call
	result := c.db.Where("sid = ?", c.callSid).First(&call)

	var clientPhone string
	if c.agent.PhoneNumber != toPhoneNumber {
		clientPhone = toPhoneNumber
	} else if c.agent.PhoneNumber != fromPhoneNumber {
		clientPhone = fromPhoneNumber
	}

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.call = &models.Call{
				AgentId: c.agent.ID,
				Sid:     c.callSid,
				ClientNumber: clientPhone,
			}

			// Set the client number based on the phone number not associated with the agent
			c.db.Create(c.call)
		} else {
			return result.Error
		}
	} else {
		// Call already exists, update
		c.call = &call
		c.call.ClientNumber = clientPhone
		c.db.Save(c.call)
	}

	return nil
}

func (c *CallOrchestrator) fetchCall() (*twilioApi.ApiV2010Call, error) {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: c.cfg.TwilioAccountSid,
		Password: c.cfg.TwilioAccountAuthToken,
	})

	// Fetch the call using the call SID
	call, err := client.Api.FetchCall(c.callSid, &twilioApi.FetchCallParams{
		PathAccountSid: &c.cfg.TwilioAccountSid,
	})

	return call, err
}

func (c *CallOrchestrator) setAgent(call *twilioApi.ApiV2010Call) error {
	toPhoneNumber := call.To
	fromPhoneNumber := call.From

	// Look up the agent using the "to" or "from" phone number
	var agent models.Agent
	result := c.db.Where("phone_number = ? OR phone_number = ?", *toPhoneNumber, *fromPhoneNumber).First(&agent)
	if result.Error != nil {
		return result.Error
	}

	c.agent = &agent

	return nil
}
