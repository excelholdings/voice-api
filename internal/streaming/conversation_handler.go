package streaming

import (
	"context"
	"fmt"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/sashabaranov/go-openai"
	"time"
)

type LLM struct {
	BaseURL string
	Model   string
	APIKey  string
}

func (c *CallOrchestrator) getLLM(model string) LLM {
	llms := map[string]LLM{
		"gpt-4o": {
			BaseURL: "https://api.openai.com/v1",
			Model: "gpt-4o",
			APIKey: c.cfg.OpenAIAPIKey,
		},
		"flyflow-voice": {
			BaseURL: "https://api.fireworks.ai/inference/v1",
			Model: "accounts/fireworks/models/llama-v3-70b-instruct",
			APIKey: c.cfg.FireworksAPIKey,
		},
	}
	llm, ok := llms[model]
	if !ok {
		return llms["gpt-4o"]
	}
	return llm
}

func (c *CallOrchestrator) handleLLM() {
	llm := c.getLLM(c.agent.LLMModel)
	openaiConfig := openai.DefaultConfig(llm.APIKey)
	openaiConfig.BaseURL = llm.BaseURL
	openaiClient := openai.NewClientWithConfig(openaiConfig)

	c.call.Transcript = append(c.call.Transcript, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: c.agent.SystemPrompt,
	})

	if c.agent.InitialMessage != "" {
		c.call.Transcript = append(c.call.Transcript, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: c.agent.InitialMessage,
		})
	}

	threshold := c.agent.SmartEndpointingThreshold
	if threshold == 0 {
		threshold = 70
	}

	transcript := ""
	previousFillerWord := ""

	for {
		if c.done {
			break
		}

		// Ignore any messages when it's the assistants turn to talk
		if c.turn == "assistant" {
			continue
		}

		if transcript == "" {
			transcript, _ = <-c.transcriptionsChan
			c.call.Transcript = append(c.call.Transcript, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleUser,
				Content: transcript,
			})
		}

		c.generatingText = true

		ctx := context.Background()

		c.call.Transcript[0].Content = fmt.Sprintf("%s \n\nExtra Context \n\n %s", c.agent.SystemPrompt, c.call.Context)

		probabilityChan := make(chan uint)
		go c.smartEndpointing(c.call.Transcript[1:], probabilityChan)

		completionChan := make(chan string)
		go func() {
			resp, err := openaiClient.CreateChatCompletion(
				ctx,
				openai.ChatCompletionRequest{
					Model:    llm.Model,
					Messages: c.call.Transcript,
				},
			)
			if err != nil {
				logger.S.Error("error getting openai response ", err)
				completionChan <- ""
			}

			completionChan <- resp.Choices[0].Message.Content
		}()


		probability := <-probabilityChan
		logger.S.Infof("Smart endpointing probability: %d", probability)

		if probability >= threshold {
			c.turn = "assistant"
			if c.agent.FillerWords {
				fillerWord := c.classifier.GetFillerWord(transcript, c.agent.FillerWordsWhitelist, previousFillerWord)

				previousFillerWord = fillerWord
				if fillerWord != "" {
					c.responseChan <- fillerWord
				}
			}
			fullMessage := <- completionChan
			c.responseChan <- fullMessage
			c.call.Transcript = append(c.call.Transcript, openai.ChatCompletionMessage{
				Role:    openai.ChatMessageRoleAssistant,
				Content: fullMessage,
			})
			transcript = ""
		} else {
			timer := time.NewTimer(calcBackoff(threshold, probability, transcript))
			select {
			case newTranscript := <-c.transcriptionsChan:
				logger.S.Infof("new transcript: %v", newTranscript)
				c.call.Transcript = append(c.call.Transcript, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleUser,
					Content: newTranscript,
				})
				timer.Stop()
			case <-timer.C:
				c.turn = "assistant"
				if c.agent.FillerWords {
					fillerWord := c.classifier.GetFillerWord(transcript, c.agent.FillerWordsWhitelist, previousFillerWord)

					previousFillerWord = fillerWord
					if fillerWord != "" {
						c.responseChan <- fillerWord
					}
				}
				fullMessage := <- completionChan
				c.responseChan <- fullMessage
				c.call.Transcript = append(c.call.Transcript, openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: fullMessage,
				})
				transcript = ""
			}
		}

		c.generatingText = false
	}
}

func calcBackoff(threshold uint, probability uint, userSentence string) time.Duration {
	// Calculate the initial backoff
	backoff := time.Duration(500*(threshold-probability)/10) * time.Millisecond

	return backoff
}