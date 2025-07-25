package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/flyflow-devs/flyflow/internal/languages"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/flyflow-devs/flyflow/internal/models"
	"github.com/flyflow-devs/flyflow/internal/slack"
	"github.com/flyflow-devs/flyflow/internal/voices"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"time"
)

func (a *API) UpsertAgent(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse the request body
	var agentReq models.Agent
	err = json.NewDecoder(r.Body).Decode(&agentReq)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	slack.PostMessage(fmt.Sprintf("%s created or updated agent %s", user.Email, agentReq.Name))

	// Validate the LLM model
	if agentReq.LLMModel != "gpt-4o" && agentReq.LLMModel != "flyflow-voice" && agentReq.LLMModel != "" {
		http.Error(w, "Invalid request payload, model must be either gpt-4o, flyflow-voice or unset", http.StatusBadRequest)
		return
	}

	if agentReq.LLMModel == "" {
		agentReq.LLMModel = "gpt-4o"
	}

	for _, check := range agentReq.ComplianceChecks {
		if check.Model != "gpt-4o" && check.Model != "gpt-4-turbo" && check.Model != "gpt-3.5-turbo" {
			http.Error(w, "Invalid request payload, compliance check payload must be gpt-4o or gpt-4-turbo or gpt-3.5-turbo", http.StatusBadRequest)
			return
		}
	}

	_, ok := voices.Voices[agentReq.VoiceId]
	_, cartesiaOk := voices.CartesiaVoices[agentReq.VoiceId]
	if (!ok && !cartesiaOk) || agentReq.VoiceId == "" {
		agentReq.VoiceId = "female-young-american-warm"
	}

	if _, ok := languages.Languages[agentReq.Language]; !ok {
		http.Error(w, "Invalid request payload, model must be valid language choice: https://docs.flyflow.dev/docs/multilingual-agents", http.StatusBadRequest)
		return
	}

	// Find the existing agent based on the user ID and agent name
	var existingAgent models.Agent
	result := a.DB.Where("user_id = ? AND name = ?", user.ID, agentReq.Name).First(&existingAgent)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Agent doesn't exist, create a new one
			newAgent := &models.Agent{
				UserId:              user.ID,
				Name:                agentReq.Name,
				SystemPrompt:        agentReq.SystemPrompt,
				InitialMessage:      agentReq.InitialMessage,
				LLMModel:            agentReq.LLMModel,
				VoiceId:             agentReq.VoiceId,
				Webhook:             agentReq.Webhook,
				Tools:               agentReq.Tools,
				FillerWords:         agentReq.FillerWords,
				Actions:             agentReq.Actions,
				VoicemailNumber:     agentReq.VoicemailNumber,
				Chunking:            agentReq.Chunking,
				Endpointing:         agentReq.Endpointing,
				VoiceOptimization:   agentReq.VoiceOptimization,
				FillerWordsWhitelist: agentReq.FillerWordsWhitelist,
				SmartEndpointingThreshold: agentReq.SmartEndpointingThreshold,
				Multilingual: agentReq.Multilingual,
				Language: agentReq.Language,
				ComplianceChecks: agentReq.ComplianceChecks,
			}

			// Create a new Twilio client
			client := twilio.NewRestClientWithParams(twilio.ClientParams{
				Username: a.Cfg.TwilioAccountSid,
				Password: a.Cfg.TwilioAccountAuthToken,
			})

			// Buy a new phone number for the agent
			params := &openapi.CreateIncomingPhoneNumberParams{}
			params.SetAreaCode(agentReq.AreaCode)
			resp, err := client.Api.CreateIncomingPhoneNumber(params)
			if err != nil {
				logger.S.Warn(err)
				http.Error(w, "Failed to buy phone number, try a different area code", http.StatusBadRequest)
				return
			}

			// Set the phone number and Twilio phone SID for the new agent
			newAgent.PhoneNumber = *resp.PhoneNumber
			newAgent.TwilioPhoneSid = *resp.Sid

			// Update the webhook URL for the new phone number
			params2 := &openapi.UpdateIncomingPhoneNumberParams{}
			params2.SetVoiceUrl(a.Cfg.TwilioMLUrl)
			params2.SetVoiceMethod("POST")
			params2.SetVoiceApplicationSid(a.Cfg.TwilioMLSid)
			_, err = client.Api.UpdateIncomingPhoneNumber(*resp.Sid, params2)
			if err != nil {
				logger.S.Error(err)
				http.Error(w, "Failed to update phone number", http.StatusInternalServerError)
				return
			}

			// Save the new agent in the database
			result = a.DB.Create(newAgent)
			if result.Error != nil {
				logger.S.Error(result.Error)
				http.Error(w, "Failed to create agent", http.StatusInternalServerError)
				return
			}

			// Return the new agent object
			json.NewEncoder(w).Encode(newAgent)
		} else {
			logger.S.Error(result.Error)
			http.Error(w, "Failed to retrieve agent", http.StatusInternalServerError)
			return
		}
	} else {
		// Agent exists, update all the fields based on the request
		existingAgent.SystemPrompt = agentReq.SystemPrompt
		existingAgent.InitialMessage = agentReq.InitialMessage
		existingAgent.LLMModel = agentReq.LLMModel
		existingAgent.VoiceId = agentReq.VoiceId
		existingAgent.Webhook = agentReq.Webhook
		existingAgent.Tools = agentReq.Tools
		existingAgent.FillerWords = agentReq.FillerWords
		existingAgent.Actions = agentReq.Actions
		existingAgent.VoicemailNumber = agentReq.VoicemailNumber
		existingAgent.Chunking = agentReq.Chunking
		existingAgent.Endpointing = agentReq.Endpointing
		existingAgent.VoiceOptimization = agentReq.VoiceOptimization
		existingAgent.FillerWordsWhitelist = agentReq.FillerWordsWhitelist
		existingAgent.SmartEndpointingThreshold = agentReq.SmartEndpointingThreshold
		existingAgent.Multilingual = agentReq.Multilingual
		existingAgent.Language = agentReq.Language
		existingAgent.ComplianceChecks = agentReq.ComplianceChecks

		// Save the updated agent in the database
		result = a.DB.Save(&existingAgent)
		if result.Error != nil {
			logger.S.Error(result.Error)
			http.Error(w, "Failed to update agent", http.StatusInternalServerError)
			return
		}

		// Return the updated agent object
		json.NewEncoder(w).Encode(existingAgent)
	}
}


func (a *API) GetAgent(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get the agent ID from the URL parameters
	agentID := r.URL.Query().Get("id")
	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Retrieve the agent from the database
	var agent models.Agent
	result := a.DB.Where("id = ? AND user_id = ?", agentID, user.ID).First(&agent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "Agent not found", http.StatusNotFound)
		} else {
			logger.S.Error(result.Error)
			http.Error(w, "Failed to retrieve agent", http.StatusInternalServerError)
		}
		return
	}

	// Return the agent object
	json.NewEncoder(w).Encode(agent)
}

func (a *API) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get the agent ID from the URL parameters
	agentID := r.URL.Query().Get("id")
	if agentID == "" {
		http.Error(w, "Agent ID is required", http.StatusBadRequest)
		return
	}

	// Delete the agent from the database
	result := a.DB.Where("id = ? AND user_id = ?", agentID, user.ID).Delete(&models.Agent{})
	if result.Error != nil {
		logger.S.Error(result.Error)
		http.Error(w, "Failed to delete agent", http.StatusInternalServerError)
		return
	}

	// Check if the agent was deleted
	if result.RowsAffected == 0 {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	// Return a success response
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) ListAgents(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse the query parameters
	queryParams := r.URL.Query()
	cursor := queryParams.Get("cursor")
	limit := queryParams.Get("limit")
	if limit == "" {
		limit = "10"
	}

	// Convert limit to integer
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
		return
	}

	// Create a base query
	query := a.DB.Model(&models.Agent{}).
		Where("user_id = ?", user.ID).
		Order("created_at DESC").
		Limit(limitInt + 1)

	// Apply cursor if provided
	if cursor != "" {
		query = query.Where("created_at < ?", cursor)
	}

	// Retrieve the agents from the database
	var agents []models.Agent
	result := query.Find(&agents)
	if result.Error != nil {
		logger.S.Error(result.Error)
		http.Error(w, "Failed to retrieve agents", http.StatusInternalServerError)
		return
	}

	// Check if there are more agents
	hasMore := len(agents) > limitInt
	if hasMore {
		agents = agents[:limitInt]
	}

	// Prepare the response
	response := struct {
		NumItems int             `json:"num_items"`
		Cursor   string          `json:"cursor,omitempty"`
		Agents   []models.Agent `json:"agents"`
	}{
		NumItems: len(agents),
		Agents:   agents,
	}

	// Set the cursor if there are more agents
	if hasMore {
		response.Cursor = agents[len(agents)-1].CreatedAt.Format(time.RFC3339)
	}

	// Return the response
	json.NewEncoder(w).Encode(response)
}