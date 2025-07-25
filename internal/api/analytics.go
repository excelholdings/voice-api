package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/flyflow-devs/flyflow/internal/models"
)

type AnalyticsResponse struct {
	InProgressCalls  int64     `json:"in_progress_calls"`
	TotalCalls       int64     `json:"total_calls"`
	TotalMinutes     float64 `json:"total_minutes"`
	AverageSentiment float64 `json:"average_sentiment"`
}

func (a *API) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var response AnalyticsResponse

	// Get in-progress calls
	if err := a.DB.Model(&models.Call{}).
		Joins("JOIN agents ON calls.agent_id = agents.id").
		Where("agents.user_id = ? AND calls.in_progress = True", user.ID).
		Count(&response.InProgressCalls).Error; err != nil {
		http.Error(w, "Failed to get in-progress calls", http.StatusInternalServerError)
		return
	}

	// Get total calls
	if err := a.DB.Model(&models.Call{}).
		Joins("JOIN agents ON calls.agent_id = agents.id").
		Where("agents.user_id = ?", user.ID).
		Count(&response.TotalCalls).Error; err != nil {
		http.Error(w, "Failed to get total calls", http.StatusInternalServerError)
		return
	}

	// Calculate total minutes this month
	startOfMonth := getStartOfMonth(user.CreatedAt)
	var totalSeconds float64
	if err := a.DB.Model(&models.Call{}).
		Joins("JOIN agents ON calls.agent_id = agents.id").
		Where("agents.user_id = ? AND calls.created_at >= ?", user.ID, startOfMonth).
		Select("SUM(time_seconds)").
		Scan(&totalSeconds).Error; err != nil {
		http.Error(w, "Failed to get total minutes", http.StatusInternalServerError)
		return
	}
	response.TotalMinutes = totalSeconds / 60 // Convert seconds to minutes

	// Calculate average sentiment
	var result struct {
		AvgSentiment float64
	}
	if err := a.DB.Model(&models.Call{}).
		Joins("JOIN agents ON calls.agent_id = agents.id").
		Where("agents.user_id = ?", user.ID).
		Select("AVG(sentiment) as avg_sentiment").
		Scan(&result).Error; err != nil {
		http.Error(w, "Failed to get average sentiment", http.StatusInternalServerError)
		return
	}
	response.AverageSentiment = result.AvgSentiment

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getStartOfMonth(date time.Time) time.Time {
	year, month, day := date.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, date.Location())
}
