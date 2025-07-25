package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/flyflow-devs/flyflow/internal/config"
)

type CartesiaClient struct {
	cfg *config.Config
}

func NewCartesiaClient(cfg *config.Config) *CartesiaClient {
	return &CartesiaClient{cfg: cfg}
}

func (c *CartesiaClient) StreamSpeechBytes(modelID, transcript string, voiceID string, language string) (io.ReadCloser, error) {
	reqBody := map[string]interface{}{
		"model_id":   modelID,
		"transcript": transcript,
		"voice": map[string]string{
			"mode": "id",
			"id":   voiceID,
		},
		"output_format": map[string]interface{}{
			"container":   "raw",
			"encoding":    "pcm_mulaw",
			"sample_rate": 8000,
		},
		"language": language,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", "https://api.cartesia.ai/tts/bytes", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.cfg.CartesiaAPIKey)
	httpReq.Header.Set("Cartesia-Version", c.cfg.CartesiaVersion)

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp.Body, nil
}