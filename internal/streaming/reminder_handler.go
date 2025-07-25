package streaming

import "time"

func (c *CallOrchestrator) handleReminders() {
	for {
		if c.done {
			break
		}

		if time.Since(c.lastFinalizedMessage) > 10 * time.Second {
			c.transcriptionsChan <- ""
			c.lastFinalizedMessage = time.Now()
		}
	}
}