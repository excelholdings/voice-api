package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/sashabaranov/go-openai"
	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
	"net/http"
	"net/url"
	"time"
)

func (c *CallOrchestrator) handleActions() {
	var messages []openai.ChatCompletionMessage

	openaiConfig := openai.DefaultConfig(c.cfg.OpenAIAPIKey)
	openaiClient := openai.NewClientWithConfig(openaiConfig)

	// Build the tools list
	tools := []openai.Tool{}
	for _, action := range c.agent.Actions {
		parameters := map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"ForwardingNumber": map[string]interface{}{
					"type":        "string",
					"description": "The phone number to forward the call to",
				},
			},
			"required": []string{"ForwardingNumber"},
		}

		tools = append(tools, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        action.Name,
				Description: fmt.Sprintf("Instructions: %v \n\nForwarding Number: %v", action.Instructions, action.ForwardingNumber),
				Parameters:  parameters,
			},
		})
	}

	for {
		if c.done {
			break
		}

		if len(c.call.Transcript) > len(messages) {
			ctx := context.Background()
			resp, err := openaiClient.CreateChatCompletion(
				ctx,
				openai.ChatCompletionRequest{
					Model:    "gpt-4",
					Messages: c.call.Transcript,
					Tools:    tools,
				},
			)
			if err != nil {
				logger.S.Errorf("error getting tool call: %v", err)
				continue
			}

			if len(resp.Choices) > 0 && len(resp.Choices[0].Message.ToolCalls) > 0 {
				for _, toolCall := range resp.Choices[0].Message.ToolCalls {
					switch toolCall.Function.Name {
					case "hangup":
						if err := c.hangupCall(); err != nil {
							logger.S.Errorf("error hanging up call: %v", err)
						}
					case "forward":
						var args struct {
							ForwardingNumber string `json:"ForwardingNumber"`
						}
						if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
							logger.S.Errorf("error parsing forward arguments: %v", err)
							continue
						}
						fmt.Printf(args.ForwardingNumber)
						if err := c.forwardCall(args.ForwardingNumber); err != nil {
							logger.S.Errorf("error forwarding call: %v", err)
						}
					default:
						logger.S.Warnf("unknown action: %s", toolCall.Function.Name)
					}

					c.EmitEvent("action", toolCall)
				}
			}

			messages = c.call.Transcript
		}
	}
}

func (c *CallOrchestrator) hangupCall() error {
	// Wait until the agent has stopped speaking to forward
	for len(c.marks) > 0 {
		time.Sleep(1 * time.Second)
	}

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: c.cfg.TwilioAccountSid,
		Password: c.cfg.TwilioAccountAuthToken,
	})

	params := &openapi.UpdateCallParams{}
	params.SetStatus("completed")

	_, err := client.Api.UpdateCall(c.callSid, params)
	if err != nil {
		logger.S.Errorf("error hanging up call: %v", err)
		return err
	}

	c.call.DisconnectReason = "agent_hangup"
	c.doneChan <- true
	return nil
}

func (c *CallOrchestrator) forwardCall(forwardingNumber string) error {
	// Wait until the agent has stopped speaking to forward
	for len(c.marks) > 0 {
		time.Sleep(1 * time.Second)
	}

	c.call.DisconnectReason = "forward"

	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: c.cfg.TwilioAccountSid,
		Password: c.cfg.TwilioAccountAuthToken,
	})

	params := &openapi.UpdateCallParams{}
	forwardURL, _ := url.Parse(c.cfg.ForwardRedirectMLUrl)
	q := forwardURL.Query()
	q.Set("ForwardingNumber", forwardingNumber)
	forwardURL.RawQuery = q.Encode()
	params.SetUrl(forwardURL.String())
	params.SetMethod(http.MethodPost)

	_, err := client.Api.UpdateCall(c.callSid, params)
	if err != nil {
		logger.S.Errorf("error forwarding call: %v", err)
		return err
	}

	return nil
}