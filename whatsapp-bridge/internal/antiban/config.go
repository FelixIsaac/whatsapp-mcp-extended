package antiban

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type MessageType int

const (
	Text MessageType = iota
	Reaction
	Edit
	Delete
	Poll
	Peer
)

type Config struct {
	Enabled            bool
	TextDelayMin       time.Duration
	TextDelayMax       time.Duration
	FeedbackDelayMin   time.Duration
	FeedbackDelayMax   time.Duration
	TypingMsPerChar    int
	WarmUpDays         int
	WarmUpStartLimit   int
	WarmUpStatePath    string
	RiskPauseThreshold int
}

func LoadConfig() *Config {
	return &Config{
		Enabled:            loadBoolEnv("ANTIBAN_ENABLED", false),
		TextDelayMin:       loadDurationEnv("ANTIBAN_TEXT_DELAY_MIN", 1500*time.Millisecond),
		TextDelayMax:       loadDurationEnv("ANTIBAN_TEXT_DELAY_MAX", 4*time.Second),
		FeedbackDelayMin:   loadDurationEnv("ANTIBAN_FEEDBACK_DELAY_MIN", 500*time.Millisecond),
		FeedbackDelayMax:   loadDurationEnv("ANTIBAN_FEEDBACK_DELAY_MAX", 1500*time.Millisecond),
		TypingMsPerChar:    loadIntEnv("ANTIBAN_TYPING_MS_PER_CHAR", 30),
		WarmUpDays:         loadIntEnv("ANTIBAN_WARMUP_DAYS", 7),
		WarmUpStartLimit:   loadIntEnv("ANTIBAN_WARMUP_START_LIMIT", 20),
		WarmUpStatePath:    loadStringEnv("ANTIBAN_WARMUP_STATE_PATH", "store/antiban_warmup.json"),
		RiskPauseThreshold: loadIntEnv("ANTIBAN_RISK_PAUSE_THRESHOLD", 70),
	}
}

func (m MessageType) String() string {
	switch m {
	case Text:
		return "text"
	case Reaction:
		return "reaction"
	case Edit:
		return "edit"
	case Delete:
		return "delete"
	case Poll:
		return "poll"
	case Peer:
		return "peer"
	default:
		return fmt.Sprintf("unknown(%d)", int(m))
	}
}

func loadBoolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func loadIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func loadDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func loadStringEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
