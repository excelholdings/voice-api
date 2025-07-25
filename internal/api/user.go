package api

import (
	"encoding/json"
	"fmt"
	"github.com/flyflow-devs/flyflow/internal/models"
	"github.com/flyflow-devs/flyflow/internal/slack"
	"net/http"
)

// GetUser handles the GET request to retrieve user information
func (a *API) GetUser(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if user.Plan == "" || user.Plan == "free" {
		user.Plan = "free"
		user.Details = models.PlanDetails{
			Minutes: 50,
			PricePerMinute: 0.2,
		}
	}

	if user.Plan == "pro" {
		user.Details = models.PlanDetails{
			Minutes: 500,
			PricePerMinute: 0.15,
		}
	}

	// Exclude sensitive information
	userResponse := struct {
		ID    uint   `json:"id"`
		Email string `json:"email"`
		Plan  string `json:"plan"`
		Details models.PlanDetails `json:"details"`
	}{
		ID:    user.ID,
		Email: user.Email,
		Plan:  user.Plan,
		Details: user.Details,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userResponse)
}

// SetPlan handles the POST request to set a user's plan
func (a *API) SetPlan(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	user, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse the request body
	var planRequest struct {
		Plan    string `json:"plan"`
		Minutes uint   `json:"minutes"`
		PricePerMinute float64 `json:"price_per_minute"`
	}
	if err := json.NewDecoder(r.Body).Decode(&planRequest); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	slack.PostMessage(fmt.Sprintf("%s changed plan to %s", user.Email, planRequest.Plan))

	// Update the user's plan and details
	user.Plan = planRequest.Plan
	user.Details = models.PlanDetails{
		Minutes:        planRequest.Minutes,
		PricePerMinute: planRequest.PricePerMinute,
	}

	// Save the updated user to the database
	if err := a.DB.Save(user).Error; err != nil {
		http.Error(w, "Failed to update user plan", http.StatusInternalServerError)
		return
	}

	// Return the updated user information
	userResponse := struct {
		ID    uint   `json:"id"`
		Email string `json:"email"`
		Plan  string `json:"plan"`
		Details models.PlanDetails `json:"details"`
	}{
		ID:    user.ID,
		Email: user.Email,
		Plan:  user.Plan,
		Details: user.Details,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userResponse)
}