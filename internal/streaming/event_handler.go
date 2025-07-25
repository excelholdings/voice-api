package streaming

import "github.com/flyflow-devs/flyflow/internal/webhook"

func (c *CallOrchestrator) EmitEvent(name string, data interface{}) {
	if c.agent.Webhook != "" {
		webhook.EmitEvent(c.agent.Webhook, name, c.call, data)
	}
}
