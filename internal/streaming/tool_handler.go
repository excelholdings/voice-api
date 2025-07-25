package streaming

import (
	"context"
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/sashabaranov/go-openai"
)

func (c *CallOrchestrator) handleToolCalls() {
	if len(c.agent.Tools) == 0 {
		logger.S.Info("no tools found, exiting")
		return
	}

	var messages []openai.ChatCompletionMessage

	openaiConfig := openai.DefaultConfig(c.cfg.OpenAIAPIKey)
	openaiClient := openai.NewClientWithConfig(openaiConfig)

	for {
		if c.done {
			break
		}

		if len(c.call.Transcript) > len(messages) && c.call.Transcript[len(c.call.Transcript)-1].Role == "user" {
			ctx := context.Background()
			resp, err := openaiClient.CreateChatCompletion(
				ctx,
				openai.ChatCompletionRequest{
					Model:    "gpt-4o",
					Messages: c.call.Transcript,
					Tools: c.agent.Tools,
				},
			)
			if err != nil {
				logger.S.Errorf("error getting tool call: %v", err)
				continue
			}

			if len(resp.Choices[0].Message.ToolCalls) > 0 {
				c.EmitEvent("tool_call", resp.Choices[0].Message.ToolCalls)
			}

			messages = c.call.Transcript
		}

	}
}
