package api

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/flyflow-devs/flyflow/internal/models"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

func (a *API) CreateCall(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse the request body
	var callReq struct {
		From    string `json:"from"`
		To      string `json:"to"`
		Context string `json:"context"`

		UserSpeaksFirst bool `json:"user_speaks_first"`
	}
	err = json.NewDecoder(r.Body).Decode(&callReq)
	if err != nil {
		logger.S.Error(err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Validate the "from" phone number
	var agent models.Agent
	result := a.DB.Where("phone_number = ? AND user_id = ?", callReq.From, user.ID).First(&agent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "Invalid 'from' phone number", http.StatusBadRequest)
		} else {
			logger.S.Error(result.Error)
			http.Error(w, "Failed to validate 'from' phone number", http.StatusInternalServerError)
		}
		return
	}

	// Create a new Twilio client
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: a.Cfg.TwilioAccountSid,
		Password: a.Cfg.TwilioAccountAuthToken,
	})

	// Make the phone call using the Twilio API
	params := &openapi.CreateCallParams{}
	params.SetTo(callReq.To)
	params.SetFrom(callReq.From)
	params.SetUrl(a.Cfg.TwilioMLUrl)
	resp, err := client.Api.CreateCall(params)
	if err != nil {
		logger.S.Error(err)
		http.Error(w, "Failed to make phone call", http.StatusInternalServerError)
		return
	}

	// Create a new Call object
	call := &models.Call{
		AgentId:    agent.ID,
		Context:    callReq.Context,
		Sid:        *resp.Sid,
		StartedAt:  time.Now(),
		UserSpeaksFirst: callReq.UserSpeaksFirst,
	}

	// Save the Call object in the database
	result = a.DB.Create(call)
	if result.Error != nil {
		logger.S.Error(result.Error)
		http.Error(w, "Failed to create call record", http.StatusInternalServerError)
		return
	}

	// Return the Call object
	json.NewEncoder(w).Encode(call)
}

func (a *API) GetCall(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get the call ID from the URL parameters
	callID := r.URL.Query().Get("id")
	if callID == "" {
		http.Error(w, "Call ID is required", http.StatusBadRequest)
		return
	}

	// Retrieve the call from the database
	var call models.Call
	result := a.DB.Where("id = ? AND agent_id IN (SELECT id FROM agents WHERE user_id = ?)", callID, user.ID).First(&call)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "Call not found", http.StatusNotFound)
		} else {
			logger.S.Error(result.Error)
			http.Error(w, "Failed to retrieve call", http.StatusInternalServerError)
		}
		return
	}

	// Return the call object
	json.NewEncoder(w).Encode(call)
}

func (a *API) SetCallContext(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse the request body
	var contextReq struct {
		ID      uint   `json:"id"`
		Context string `json:"context"`
	}
	err = json.NewDecoder(r.Body).Decode(&contextReq)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Retrieve the call from the database
	var call models.Call
	result := a.DB.Where("id = ? AND agent_id IN (SELECT id FROM agents WHERE user_id = ?)", contextReq.ID, user.ID).First(&call)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "Call not found", http.StatusNotFound)
		} else {
			logger.S.Error(result.Error)
			http.Error(w, "Failed to retrieve call", http.StatusInternalServerError)
		}
		return
	}

	// Update the call context
	call.Context = contextReq.Context
	result = a.DB.Save(&call)
	if result.Error != nil {
		logger.S.Error(result.Error)
		http.Error(w, "Failed to update call context", http.StatusInternalServerError)
		return
	}

	// Return the updated call object
	json.NewEncoder(w).Encode(call)
}

func (a *API) ListCalls(w http.ResponseWriter, r *http.Request) {
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
	agentID := queryParams.Get("agent_id")
	clientNumber := queryParams.Get("client_number")

	// Convert limit to integer
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
		return
	}

	// Create a base query
	query := a.DB.Model(&models.Call{}).
		Where("agent_id IN (SELECT id FROM agents WHERE user_id = ?)", user.ID).
		Order("created_at DESC").
		Limit(limitInt + 1)

	// Apply cursor if provided
	if cursor != "" {
		query = query.Where("created_at < ?", cursor)
	}

	// Apply agent_id filter if provided
	if agentID != "" {
		query = query.Where("agent_id = ?", agentID)
	}

	// Apply client_number filter if provided
	if clientNumber != "" {
		query = query.Where("client_number = ?", clientNumber)
	}

	// Retrieve the calls from the database
	var calls []models.Call
	result := query.Find(&calls)
	if result.Error != nil {
		logger.S.Error(result.Error)
		http.Error(w, "Failed to retrieve calls", http.StatusInternalServerError)
		return
	}

	// Check if there are more calls
	hasMore := len(calls) > limitInt
	if hasMore {
		calls = calls[:limitInt]
	}

	// Prepare the response
	response := struct {
		NumItems int           `json:"num_items"`
		Cursor   string        `json:"cursor,omitempty"`
		Calls    []models.Call `json:"calls"`
	}{
		NumItems: len(calls),
		Calls:    calls,
	}

	// Set the cursor if there are more calls
	if hasMore {
		response.Cursor = calls[len(calls)-1].CreatedAt.Format(time.RFC3339)
	}

	// Return the response
	json.NewEncoder(w).Encode(response)
}

func (a *API) GetRecording(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get the recording ID from the URL parameters
	vars := mux.Vars(r)
	recordingID := vars["id"]
	if recordingID == "" {
		http.Error(w, "Recording ID is required", http.StatusBadRequest)
		return
	}

	// Retrieve the call from the database to validate ownership
	var call models.Call
	result := a.DB.Where("id = ? AND agent_id IN (SELECT id FROM agents WHERE user_id = ?)", recordingID, user.ID).First(&call)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "Recording not found", http.StatusNotFound)
		} else {
			logger.S.Error(result.Error)
			http.Error(w, "Failed to retrieve recording", http.StatusInternalServerError)
		}
		return
	}

	// Fetch the recording media file from Twilio
	recordingURI := "https://api.twilio.com/2010-04-01/Accounts/" + a.Cfg.TwilioAccountSid + "/Recordings/" + call.RecordingSid + ".mp3"
	resp, err := downloadRecording(recordingURI, a.Cfg.TwilioAccountSid, a.Cfg.TwilioAccountAuthToken)
	if err != nil {
		logger.S.Error(err)
		http.Error(w, "Failed to download recording", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Set the appropriate headers and copy the response body
	w.Header().Set("Content-Type", "audio/mpeg")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, resp.Body)
}

// downloadRecording is a helper function to download a recording from Twilio
func downloadRecording(url, username, password string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, password)
	client := &http.Client{}
	return client.Do(req)
}

func (a *API) DeleteCall(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get the call ID from the URL parameters
	vars := mux.Vars(r)
	callID := vars["id"]
	if callID == "" {
		http.Error(w, "Call ID is required", http.StatusBadRequest)
		return
	}

	// Retrieve the call from the database to validate ownership
	var call models.Call
	result := a.DB.Where("id = ? AND agent_id IN (SELECT id FROM agents WHERE user_id = ?)", callID, user.ID).First(&call)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "Call not found", http.StatusNotFound)
		} else {
			logger.S.Error(result.Error)
			http.Error(w, "Failed to retrieve call", http.StatusInternalServerError)
		}
		return
	}

	// Delete the call from the database
	result = a.DB.Delete(&call)
	if result.Error != nil {
		logger.S.Error(result.Error)
		http.Error(w, "Failed to delete call", http.StatusInternalServerError)
		return
	}

	// Return a success message
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Call deleted successfully"})
}
