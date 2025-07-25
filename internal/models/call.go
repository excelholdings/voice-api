package models

import (
	"github.com/sashabaranov/go-openai"
	"time"
)

type Call struct {
	BaseModel
	AgentId        uint                           `json:"agent_id" gorm:"index"`
	TimeSeconds    float64                        `json:"time_seconds"`
	UserSpeaksFirst bool                          `json:"user_speaks_first"`
	AverageLatency float64                        `json:"average_latency_ms"`
	Transcript     []openai.ChatCompletionMessage `json:"transcript" gorm:"serializer:json"`
	Context        string                         `json:"context"`
	Sid            string                         `json:"twilio_sid" gorm:"index"`
	RecordingSid   string                         `json:"recording_sid"`
	ClientNumber   string                         `json:"client_number" gorm:"index"`
	Sentiment      uint                           `json:"sentiment"`
	InProgress     bool                           `json:"in_progress"`

	StartedAt time.Time  `json:"started_at"`
	EndedAt   time.Time  `json:"ended_at"`

	DisconnectReason string `json:"disconnect_reason"`
}