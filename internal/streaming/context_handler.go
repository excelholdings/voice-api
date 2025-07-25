package streaming

import (
	"github.com/flyflow-devs/flyflow/internal/logger"
	"github.com/flyflow-devs/flyflow/internal/models"
)

func (c *CallOrchestrator) handleUpdatingContext() {
	for {
		if c.done {
			break
		}

		var callWithContext models.Call
		result := c.db.Where("sid = ?", c.call.Sid).First(&callWithContext)
		if result.Error != nil {
			logger.S.Errorf("error getting call context: %v", result.Error)
			continue
		}

		c.call.Context = callWithContext.Context
	}
}