package api

import (
	"encoding/json"
	"fmt"
	"github.com/flyflow-devs/flyflow/internal/models"
	"github.com/flyflow-devs/flyflow/internal/slack"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strings"
	"time"
)

func (a *API) SignUp(w http.ResponseWriter, r *http.Request) {
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if the email domain is blocked
	if isBlockedDomain(user.Email) {
		slack.PostMessage(fmt.Sprintf("%s email blocked", user.Email))
		http.Error(w, "Email domain not allowed", http.StatusForbidden)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	user.HashedPassword = string(hashedPassword)
	result := a.DB.Create(&user)
	if result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	token, err := a.generateJWT(user.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slack.PostMessage(fmt.Sprintf("%s signed up", user.Email))

	response := map[string]string{"token": token}
	json.NewEncoder(w).Encode(response)
}

func (a *API) Login(w http.ResponseWriter, r *http.Request) {
	var user models.User
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Check if the email domain is blocked
	if isBlockedDomain(user.Email) {
		slack.PostMessage(fmt.Sprintf("%s email blocked", user.Email))
		http.Error(w, "Email domain not allowed", http.StatusForbidden)
		return
	}

	var dbUser models.User
	result := a.DB.Where("email = ?", user.Email).First(&dbUser)
	if result.Error != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(dbUser.HashedPassword), []byte(user.Password))
	if err != nil {
		http.Error(w, "Invalid email or password", http.StatusUnauthorized)
		return
	}

	token, err := a.generateJWT(user.Email)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slack.PostMessage(fmt.Sprintf("%s logged in", user.Email))

	response := map[string]string{"token": token}
	json.NewEncoder(w).Encode(response)
}

// Helper function to check if the email domain is blocked
func isBlockedDomain(email string) bool {
	blockedDomains := []string{"sugahommatreats.com", "urbanvisionmg.com"}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(parts[1])
	for _, blockedDomain := range blockedDomains {
		if domain == blockedDomain {
			return true
		}
	}
	return false
}

func (a *API) AuthCheck(w http.ResponseWriter, r *http.Request) {
	_, err := a.ValidateAuth(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (a *API) generateJWT(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(24 * time.Hour).Unix(),
	})

	return token.SignedString([]byte(a.Cfg.JWTSecret))
}