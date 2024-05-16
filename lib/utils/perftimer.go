package utils

import "time"

type PerfTimer struct {
	start    time.Time
	duration time.Duration
}

func (m *PerfTimer) Start() {
	m.start = time.Now()
}

func (m *PerfTimer) Stop() {
	m.duration = time.Since(m.start)
}

func (m *PerfTimer) ElapsedMs() float64 {
	return float64(m.duration.Abs().Microseconds()) / 1000
}
