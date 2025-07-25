package streaming

import "time"

type Metrics struct {
	startedAt time.Time
	Latencies []float64
	processed bool
}

func NewMetrics() *Metrics {
	return &Metrics{
		processed: false,
	}
}

func (m *Metrics) startProcessing() {
	m.startedAt = time.Now()
	m.processed = false
}

func (m *Metrics) stopProcessing() {
	if !m.processed {
		latency := time.Since(m.startedAt).Seconds() * 1000
		m.Latencies = append(m.Latencies, latency)
	}
	m.processed = true
}

func (m *Metrics) getAverageLatency() float64 {
	if len(m.Latencies) == 0 {
		return 0
	}

	var sum float64
	for _, latency := range m.Latencies {
		sum += latency
	}

	return sum / float64(len(m.Latencies))
}