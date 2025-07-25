package api

import (
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/flyflow-devs/flyflow/internal/models"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
)

var uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (a *API) ValidateAuth(r *http.Request) (*models.User, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("missing authorization header")
	}

	bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
	if bearerToken == authHeader {
		return nil, errors.New("invalid authorization header format")
	}

	// Check if it's an API key (UUID format) or a JWT
	if uuidRegex.MatchString(bearerToken) {
		// API key authentication
		var apiKey models.APIKey
		result := a.DB.Where("key = ?", bearerToken).First(&apiKey)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, errors.New("invalid API key")
			}
			return nil, result.Error
		}

		var user models.User
		result = a.DB.First(&user, apiKey.UserId)
		if result.Error != nil {
			return nil, errors.New("user not found")
		}

		return &user, nil
	} else {
		// JWT authentication
		token, err := jwt.Parse(bearerToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(a.Cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return nil, errors.New("invalid token")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, errors.New("invalid token claims")
		}

		email, ok := claims["email"].(string)
		if !ok {
			return nil, errors.New("invalid token claims")
		}

		var user models.User
		result := a.DB.Where("email = ?", email).First(&user)
		if result.Error != nil {
			return nil, errors.New("user not found")
		}

		return &user, nil
	}
}