package streaming

import "github.com/flyflow-devs/flyflow/internal/logger"

func (c *CallOrchestrator) handleInterruption() {
	for {
		if c.done {
			break
		}

		<- c.interruptionChan

		// Clear audio marks
		c.marks = make(map[string]interface{})

		message := TwilioMessage{
			Event:     "clear",
			StreamSid: c.streamSid,
		}

		// Write the Twilio message to the connection
		c.outgoingWebsocketLock.Lock()
		if err := c.conn.WriteJSON(message); err != nil {
			logger.S.Errorf("Error writing Twilio message: %v", err)
			continue
		}
		c.outgoingWebsocketLock.Unlock()
	}
}