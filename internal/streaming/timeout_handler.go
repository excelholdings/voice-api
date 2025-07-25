package streaming

import "time"

func (c *CallOrchestrator) handleTimeouts() {
	for {
		if c.done {
			break
		}

		if time.Since(c.userLastSpoke).Seconds() > 60 {
			c.call.DisconnectReason = "call_timeout"
			c.doneChan <- true
		}
	}
}