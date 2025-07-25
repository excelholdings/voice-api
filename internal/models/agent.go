package models

import "github.com/sashabaranov/go-openai"

type Agent struct {
	BaseModel
	UserId         uint   `json:"user_id" gorm:"index"`
	Name           string `json:"name"`
	PhoneNumber    string `json:"phone_number" gorm:"index"`
	TwilioPhoneSid string `json:"phone_sid" gorm:"index"`
	SystemPrompt   string `json:"system_prompt"`
	InitialMessage string `json:"initial_message"`
	LLMModel       string `json:"llm_model"`
	VoiceId        string `json:"voice_id"`
	Webhook        string `json:"webhook"`
	Tools          []openai.Tool `json:"tools" gorm:"serializer:json"`
	FillerWords    bool          `json:"filler_words"`
	Actions        []Action      `json:"actions" gorm:"serializer:json"`
	VoicemailNumber string       `json:"voicemail_number"`
	Chunking        bool         `json:"chunking"`
	Endpointing     uint         `json:"endpointing"`
	SmartEndpointingThreshold uint `json:"smart_endpointing_threshold"`
	VoiceOptimization uint       `json:"voice_optimization"`
	Multilingual      bool       `json:"multilingual"`
	Language          string     `json:"language"`
	ComplianceChecks  []ComplianceCheck `json:"compliance_checks" gorm:"serializer:json"`

	FillerWordsWhitelist []string `json:"filler_words_whitelist" gorm:"serializer:json"`

	AreaCode       string `json:"-" gorm:"-"`
}

type Action struct {
	Name string `json:"name"`
	Instructions string `json:"instructions"`
	ForwardingNumber string `json:"forwarding_number,omitempty"`
}

type ComplianceCheck struct {
	Name string `json:"name"`
	Model string `json:"model"`
	CheckInstructions string `json:"check_instructions"`
	RewriteThreshold uint `json:"rewrite_threshold"`
}
