package antiban

import (
	"context"
	"errors"
	"strings"
)

var ErrWarmUpLimitReached = errors.New("antiban: daily warm-up limit reached")

type SendInterceptor struct {
	cfg         *Config
	rateLimiter *RateLimiter
	warmUp      *WarmUp
	health      *HealthMonitor
}

func NewSendInterceptor(cfg *Config) (*SendInterceptor, error) {
	if cfg == nil {
		cfg = LoadConfig()
	}

	interceptor := &SendInterceptor{cfg: cfg}
	if !cfg.Enabled {
		return interceptor, nil
	}

	interceptor.rateLimiter = NewRateLimiter(cfg)

	warmUp, err := NewWarmUp(cfg)
	if err != nil {
		return nil, err
	}
	interceptor.warmUp = warmUp
	interceptor.health = NewHealthMonitor(cfg.RiskPauseThreshold)

	return interceptor, nil
}

func (s *SendInterceptor) BeforeSend(ctx context.Context, msgType MessageType, msgLen int) error {
	if !s.hasInternals() {
		return nil
	}
	if s.health.IsPaused() {
		return ErrHealthPaused
	}

	canSend, err := s.warmUp.CanSend()
	if err != nil {
		return err
	}
	if !canSend {
		return ErrWarmUpLimitReached
	}

	return s.rateLimiter.Wait(ctx, msgType, msgLen)
}

func (s *SendInterceptor) AfterSend(_ MessageType) {
	if !s.hasInternals() {
		return
	}

	s.warmUp.RecordSend()
	s.health.RecordEvent(EventSendSuccess)
}

func (s *SendInterceptor) AfterSendFailed(_ MessageType, err error) {
	if !s.hasInternals() {
		return
	}

	if isSendFailed403(err) {
		s.health.RecordEvent(EventSendFailed403)
		return
	}

	s.health.RecordEvent(EventSendFailed)
}

func (s *SendInterceptor) RecordEvent(eventType HealthEventType) {
	if !s.hasInternals() {
		return
	}

	s.health.RecordEvent(eventType)
}

func (s *SendInterceptor) Close() error {
	if !s.hasInternals() {
		return nil
	}

	err := s.warmUp.saveState()
	s.health.Close()

	return err
}

func (s *SendInterceptor) Enabled() bool {
	return s != nil && s.cfg != nil && s.cfg.Enabled
}

func (s *SendInterceptor) hasInternals() bool {
	return s != nil &&
		s.Enabled() &&
		s.rateLimiter != nil &&
		s.warmUp != nil &&
		s.health != nil
}

func isSendFailed403(err error) bool {
	if err == nil {
		return false
	}

	message := err.Error()
	return strings.Contains(message, "403") || strings.Contains(message, "401")
}
