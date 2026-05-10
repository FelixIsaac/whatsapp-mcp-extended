package antiban

import (
	"errors"
	"log"
	"sync"
	"time"
)

type HealthEventType int

const (
	EventDisconnected HealthEventType = iota
	EventKeepAliveTimeout
	EventStreamError
	EventLoggedOut
	EventSendFailed403
	EventSendFailed
	EventConnected
	EventSendSuccess
)

const (
	defaultPauseThreshold  = 70
	defaultResumeThreshold = 30
	defaultDecayInterval   = 5 * time.Minute
	healthDecayStep        = 3
)

var ErrHealthPaused = errors.New("antiban: health monitor paused sends (risk score too high)")

type HealthMonitor struct {
	score           int
	paused          bool
	mu              sync.Mutex
	pauseThreshold  int
	resumeThreshold int
	stopDecay       chan struct{}
	decayInterval   time.Duration
	closed          bool
}

func NewHealthMonitor(pauseThreshold int) *HealthMonitor {
	return newHealthMonitor(pauseThreshold, defaultDecayInterval)
}

func newHealthMonitor(pauseThreshold int, decayInterval time.Duration) *HealthMonitor {
	if pauseThreshold <= 0 {
		pauseThreshold = defaultPauseThreshold
	}
	if decayInterval <= 0 {
		decayInterval = defaultDecayInterval
	}

	monitor := &HealthMonitor{
		pauseThreshold:  pauseThreshold,
		resumeThreshold: defaultResumeThreshold,
		stopDecay:       make(chan struct{}),
		decayInterval:   decayInterval,
	}

	go monitor.runDecay()

	return monitor
}

func (h *HealthMonitor) RecordEvent(eventType HealthEventType) {
	if h == nil {
		return
	}

	h.applyDelta(deltaForEvent(eventType))
}

func (h *HealthMonitor) IsPaused() bool {
	if h == nil {
		return false
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	return h.paused
}

func (h *HealthMonitor) Score() int {
	if h == nil {
		return 0
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	return h.score
}

func (h *HealthMonitor) Close() {
	if h == nil {
		return
	}

	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	stop := h.stopDecay
	h.mu.Unlock()

	close(stop)
}

func (h *HealthMonitor) runDecay() {
	ticker := time.NewTicker(h.decayInterval)
	defer ticker.Stop()
	stopDecay := h.stopDecay

	for {
		select {
		case <-ticker.C:
			h.applyDelta(-healthDecayStep)
		case <-stopDecay:
			return
		}
	}
}

func (h *HealthMonitor) applyDelta(delta int) {
	h.mu.Lock()
	previousPaused := h.paused

	h.score += delta
	if h.score < 0 {
		h.score = 0
	}
	if h.score > 100 {
		h.score = 100
	}

	if h.score >= h.pauseThreshold {
		h.paused = true
	} else if h.score <= h.resumeThreshold {
		h.paused = false
	}

	score := h.score
	paused := h.paused
	pauseThreshold := h.pauseThreshold
	resumeThreshold := h.resumeThreshold
	h.mu.Unlock()

	if paused != previousPaused {
		if paused {
			log.Printf("ANTIBAN health paused score=%d pause_threshold=%d", score, pauseThreshold)
			return
		}

		log.Printf("ANTIBAN health resumed score=%d resume_threshold=%d", score, resumeThreshold)
	}
}

func deltaForEvent(eventType HealthEventType) int {
	switch eventType {
	case EventDisconnected:
		return 15
	case EventKeepAliveTimeout:
		return 10
	case EventStreamError:
		return 20
	case EventLoggedOut:
		return 50
	case EventSendFailed403:
		return 25
	case EventSendFailed:
		return 5
	case EventConnected:
		return -20
	case EventSendSuccess:
		return -1
	default:
		return 0
	}
}
