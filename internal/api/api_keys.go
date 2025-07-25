package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/flyflow-devs/flyflow/internal/models"
	"github.com/flyflow-devs/flyflow/internal/slack"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"time"
)

func (a *API) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var apiKeyReq struct {
		Name string `json:"name"`
	}
	err = json.NewDecoder(r.Body).Decode(&apiKeyReq)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	slack.PostMessage(fmt.Sprintf("%s created api key %s", user.Email, apiKeyReq.Name))

	apiKey := models.APIKey{
		UserId: user.ID,
		Name:   apiKeyReq.Name,
		Key:    uuid.New().String(),
	}

	result := a.DB.Create(&apiKey)
	if result.Error != nil {
		logger.S.Error(result.Error)
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(apiKey)
}

func (a *API) GetAPIKey(w http.ResponseWriter, r *http.Request) {
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	apiKeyID := r.URL.Query().Get("id")
	if apiKeyID == "" {
		http.Error(w, "API Key ID is required", http.StatusBadRequest)
		return
	}

	var apiKey models.APIKey
	result := a.DB.Where("id = ? AND user_id = ?", apiKeyID, user.ID).First(&apiKey)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "API Key not found", http.StatusNotFound)
		} else {
			logger.S.Error(result.Error)
			http.Error(w, "Failed to retrieve API Key", http.StatusInternalServerError)
		}
		return
	}

	json.NewEncoder(w).Encode(apiKey)
}

func (a *API) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	apiKeyValue := r.URL.Query().Get("key")
	if apiKeyValue == "" {
		http.Error(w, "API Key is required", http.StatusBadRequest)
		return
	}

	var apiKey models.APIKey
	result := a.DB.Where("key = ? AND user_id = ?", apiKeyValue, user.ID).First(&apiKey)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			http.Error(w, "API Key not found", http.StatusNotFound)
		} else {
			logger.S.Error(result.Error)
			http.Error(w, "Failed to retrieve API Key", http.StatusInternalServerError)
		}
		return
	}

	result = a.DB.Delete(&apiKey)
	if result.Error != nil {
		logger.S.Error(result.Error)
		http.Error(w, "Failed to delete API Key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (a *API) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	queryParams := r.URL.Query()
	cursor := queryParams.Get("cursor")
	limit := queryParams.Get("limit")
	if limit == "" {
		limit = "10"
	}

	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		http.Error(w, "Invalid limit parameter", http.StatusBadRequest)
		return
	}

	query := a.DB.Model(&models.APIKey{}).
		Where("user_id = ?", user.ID).
		Order("created_at DESC").
		Limit(limitInt + 1)

	if cursor != "" {
		query = query.Where("created_at < ?", cursor)
	}

	var apiKeys []models.APIKey
	result := query.Find(&apiKeys)
	if result.Error != nil {
		logger.S.Error(result.Error)
		http.Error(w, "Failed to retrieve API Keys", http.StatusInternalServerError)
		return
	}

	hasMore := len(apiKeys) > limitInt
	if hasMore {
		apiKeys = apiKeys[:limitInt]
	}

	response := struct {
		NumItems int              `json:"num_items"`
		Cursor   string           `json:"cursor,omitempty"`
		APIKeys  []models.APIKey `json:"api_keys"`
	}{
		NumItems: len(apiKeys),
		APIKeys:  apiKeys,
	}

	if hasMore {
		response.Cursor = apiKeys[len(apiKeys)-1].CreatedAt.Format(time.RFC3339)
	}

	json.NewEncoder(w).Encode(response)
}