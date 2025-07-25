package api

import (
	"encoding/json"
	"net/http"
)

func (a *API) GetFillerWords(w http.ResponseWriter, r *http.Request) {
	// Validate the API key
	_, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get the call ID from the URL parameters
	text := r.URL.Query().Get("text")
	if text == "" {
		http.Error(w, "Text is required", http.StatusBadRequest)
		return
	}

	probabilities, word := a.Classifier.GetProbabilities(text)

	response := struct {
		RecommendedWord string             `json:"recommended_word"`
		Probabilities   map[string]float64 `json:"probabilities"`
	}{
		RecommendedWord: word,
		Probabilities: probabilities,
	}

	// Return the call object
	json.NewEncoder(w).Encode(response)
}