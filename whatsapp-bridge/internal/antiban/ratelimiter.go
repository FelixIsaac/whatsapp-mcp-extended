package antiban

import (
	"context"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"
)

type RateLimiter struct {
	cfg      *Config
	lastSend time.Time
	mu       sync.Mutex
	rng      *rand.Rand
}

func NewRateLimiter(cfg *Config) *RateLimiter {
	if cfg == nil {
		cfg = LoadConfig()
	}
	return &RateLimiter{
		cfg: cfg,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (r *RateLimiter) Wait(ctx context.Context, msgType MessageType, msgLen int) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()

	if !r.cfg.Enabled {
		r.lastSend = time.Now()
		r.mu.Unlock()
		return nil
	}

	delay := r.computeDelay(msgType, msgLen)

	if !r.lastSend.IsZero() {
		delay -= time.Since(r.lastSend)
	}

	if delay < 0 {
		delay = 0
	}

	r.lastSend = time.Now().Add(delay)
	r.mu.Unlock()

	if delay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	log.Printf("ANTIBAN delay=%s type=%s", delay, msgType)

	return nil
}

func (r *RateLimiter) computeDelay(msgType MessageType, msgLen int) time.Duration {
	var minDelay time.Duration
	var maxDelay time.Duration

	switch msgType {
	case Text, Poll:
		minDelay = r.cfg.TextDelayMin
		maxDelay = r.cfg.TextDelayMax

		if msgLen > 0 && r.cfg.TypingMsPerChar > 0 {
			typingDelay := time.Duration(msgLen*r.cfg.TypingMsPerChar) * time.Millisecond
			if typingDelay > 3*time.Second {
				typingDelay = 3 * time.Second
			}
			minDelay += typingDelay
			maxDelay += typingDelay
		}
	case Reaction, Edit, Delete:
		minDelay = r.cfg.FeedbackDelayMin
		maxDelay = r.cfg.FeedbackDelayMax
	case Peer:
		maxDelay = 500 * time.Millisecond
	default:
		minDelay = r.cfg.TextDelayMin
		maxDelay = r.cfg.TextDelayMax
	}

	if maxDelay < minDelay {
		minDelay, maxDelay = maxDelay, minDelay
	}

	if maxDelay <= minDelay {
		return minDelay
	}

	midpoint := float64(minDelay+maxDelay) / 2
	stddev := float64(maxDelay-minDelay) / 4
	value := midpoint + r.rng.NormFloat64()*stddev

	if value < float64(minDelay) {
		value = float64(minDelay)
	}
	if value > float64(maxDelay) {
		value = float64(maxDelay)
	}

	return time.Duration(math.Round(value))
}
