package streaming

import (
	"github.com/flyflow-devs/flyflow/internal/classifier"
	"github.com/flyflow-devs/flyflow/internal/config"
	"github.com/flyflow-devs/flyflow/internal/models"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"net/http"
	"sync"

	"github.com/twilio/twilio-go/twiml"
)

type TwilioHandler struct {
	Cfg *config.Config
	DB *gorm.DB
	classifier *classifier.Classifier
	wg     *sync.WaitGroup
}

func NewTwilioHandler(cfg *config.Config, db *gorm.DB, wg *sync.WaitGroup) *TwilioHandler {
	return &TwilioHandler{
		Cfg: cfg,
		DB: db,
		classifier: classifier.NewClassifier(),
		wg: wg,

	}
}

func (h *TwilioHandler) HandleTwilioML(w http.ResponseWriter, r *http.Request) {
	to := r.FormValue("To")
	from := r.FormValue("From")

	var agent models.Agent
	result := h.DB.Where("phone_number = ? OR phone_number = ?", to, from).First(&agent)
	if result.Error != nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	var elements []twiml.Element
	if agent.VoicemailNumber != "" {
		elements = append(elements, twiml.VoiceDial{
			Number: agent.VoicemailNumber,
		})
	}

	elements = append(elements, twiml.VoiceConnect{
		InnerElements: []twiml.Element{
			twiml.VoiceStream{
				Url: h.Cfg.TwilioStreamingURL,
			},
		},
	})

	resp, err := twiml.Voice(elements)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the content type to "application/xml"
	w.Header().Set("Content-Type", "application/xml")

	// Write the TwiML response to the response writer
	_, err = w.Write([]byte(resp))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *TwilioHandler) HandleTwilioStream(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := websocket.Upgrade(w, r, nil, 1024, 1024)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.wg.Add(1)
	defer h.wg.Done()

	defer conn.Close()

	// Orchestrate the call
	orchestrator := NewCallOrchestrator(h.Cfg, h.DB, conn, h.classifier)

	orchestrator.OrchestrateCall()
}

func (h *TwilioHandler) HandleForwardCall(w http.ResponseWriter, r *http.Request) {
	to := r.FormValue("To")
	from := r.FormValue("From")
	forwardingNumber := r.FormValue("ForwardingNumber")

	if forwardingNumber == "" {
		http.Error(w, "Forwarding number not provided", http.StatusBadRequest)
		return
	}

	var agent models.Agent
	result := h.DB.Where("phone_number = ? OR phone_number = ?", to, from).First(&agent)
	if result.Error != nil {
		http.Error(w, "Agent not found", http.StatusNotFound)
		return
	}

	resp, err := twiml.Voice([]twiml.Element{
		twiml.VoiceDial{
			Number: forwardingNumber,
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Set the content type to "application/xml"
	w.Header().Set("Content-Type", "application/xml")

	// Write the TwiML response to the response writer
	_, err = w.Write([]byte(resp))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}


