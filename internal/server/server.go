package server

import (
	"github.com/flyflow-devs/flyflow/internal/api"
	"github.com/flyflow-devs/flyflow/internal/config"
	"github.com/flyflow-devs/flyflow/internal/streaming"
	"gorm.io/gorm"
	"net/http"
	"sync"

	"github.com/gorilla/mux"

	"github.com/gorilla/handlers"
)

type Server struct {
	Router *mux.Router
	DB     *gorm.DB
	Cfg    *config.Config
	WG *sync.WaitGroup
}

func NewServer(Config *config.Config, DB *gorm.DB) *Server {
	s := &Server{
		Router: mux.NewRouter(),
		Cfg:    Config,
		DB:     DB,
		WG: &sync.WaitGroup{},
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	// CORS setup
	corsMiddleware := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}),
		handlers.AllowCredentials(),
	)

	// Apply CORS middleware to the entire router
	s.Router.Use(corsMiddleware)

	s.Router.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Twilio routes
	twilioHandler := streaming.NewTwilioHandler(s.Cfg, s.DB, s.WG)
	s.Router.HandleFunc("/twilio/stream", twilioHandler.HandleTwilioStream).Methods(http.MethodGet)
	s.Router.HandleFunc("/twilio/ml", twilioHandler.HandleTwilioML).Methods(http.MethodPost)
	s.Router.HandleFunc("/twilio/ml/redirect", twilioHandler.HandleForwardCall).Methods(http.MethodPost)

	// API routes
	apiHandler := api.NewAPI(s.Cfg, s.DB)
	s.Router.HandleFunc("/v1/call", apiHandler.CreateCall).Methods(http.MethodPost)
	s.Router.HandleFunc("/v1/call", apiHandler.GetCall).Methods(http.MethodGet)
	s.Router.HandleFunc("/v1/call/context", apiHandler.SetCallContext).Methods(http.MethodPost)
	s.Router.HandleFunc("/v1/calls", apiHandler.ListCalls).Methods(http.MethodGet)
	s.Router.HandleFunc("/v1/call/recording/{id}.mp3", apiHandler.GetRecording).Methods(http.MethodGet)
	s.Router.HandleFunc("/v1/call/{id}", apiHandler.DeleteCall).Methods(http.MethodDelete)

	s.Router.HandleFunc("/v1/agent", apiHandler.UpsertAgent).Methods(http.MethodPost)
	s.Router.HandleFunc("/v1/agent", apiHandler.GetAgent).Methods(http.MethodGet)
	s.Router.HandleFunc("/v1/agent", apiHandler.DeleteAgent).Methods(http.MethodDelete)
	s.Router.HandleFunc("/v1/agents", apiHandler.ListAgents).Methods(http.MethodGet)

	s.Router.HandleFunc("/v1/filler-words", apiHandler.GetFillerWords).Methods(http.MethodGet)

	// Web routes
	s.Router.HandleFunc("/v1/signup", apiHandler.SignUp).Methods(http.MethodPost)
	s.Router.HandleFunc("/v1/login", apiHandler.Login).Methods(http.MethodPost)
	s.Router.HandleFunc("/v1/authcheck", apiHandler.AuthCheck).Methods(http.MethodGet)

	s.Router.HandleFunc("/v1/apikey", apiHandler.CreateAPIKey).Methods(http.MethodPost)
	s.Router.HandleFunc("/v1/apikey", apiHandler.GetAPIKey).Methods(http.MethodGet)
	s.Router.HandleFunc("/v1/apikey", apiHandler.DeleteAPIKey).Methods(http.MethodDelete)
	s.Router.HandleFunc("/v1/apikeys", apiHandler.ListAPIKeys).Methods(http.MethodGet)

	s.Router.HandleFunc("/v1/set-payment-method", apiHandler.SetPaymentMethod).Methods(http.MethodPost)
	s.Router.HandleFunc("/v1/payment-methods", apiHandler.GetPaymentMethods).Methods(http.MethodGet, http.MethodOptions)

	s.Router.HandleFunc("/v1/analytics", apiHandler.GetAnalytics).Methods(http.MethodGet)

	s.Router.HandleFunc("/v1/user", apiHandler.GetUser).Methods(http.MethodGet)
	s.Router.HandleFunc("/v1/user/plan", apiHandler.SetPlan).Methods(http.MethodPost)
}
