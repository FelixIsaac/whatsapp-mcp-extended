package whatsapp

import (
	"math/rand"
	"sync"
	"time"
)

// Presence modes, selected via WA_PRESENCE_MODE.
const (
	// PresenceModeHuman keeps the account offline and only goes online in short
	// bursts around real outgoing activity. Default.
	PresenceModeHuman = "human"
	// PresenceModeAlwaysOnline restores the legacy behaviour: online as long as
	// the bridge is connected.
	PresenceModeAlwaysOnline = "always_online"
)

// presenceManager implements the human-like presence behaviour: the account is
// offline by default, goes online when the bridge performs a visible action
// (send, media, read receipt, reaction) and drops back offline after a short,
// slightly randomised linger without further activity.
//
// Every activity supersedes the pending linger. Instead of resetting the timer
// (which races with an already-fired callback), each activity bumps a generation
// counter and installs a fresh timer; callbacks from superseded generations are
// ignored, so no stale callback can push the client offline right after it went
// online.
type presenceManager struct {
	mu        sync.Mutex
	mode      string
	online    bool
	timer     *time.Timer
	gen       uint64
	lingerMin time.Duration
	lingerMax time.Duration
}

func newPresenceManager(mode string, lingerMin, lingerMax time.Duration) *presenceManager {
	if lingerMax < lingerMin {
		lingerMax = lingerMin
	}
	return &presenceManager{mode: mode, lingerMin: lingerMin, lingerMax: lingerMax}
}

// linger returns a randomised offline delay within the configured window.
func (pm *presenceManager) linger() time.Duration {
	spread := pm.lingerMax - pm.lingerMin
	if spread <= 0 {
		return pm.lingerMin
	}
	return pm.lingerMin + time.Duration(rand.Int63n(int64(spread)+1))
}

// PresenceMode returns the active presence mode.
func (c *Client) PresenceMode() string {
	if c.presence == nil {
		return PresenceModeAlwaysOnline
	}
	return c.presence.mode
}

// ApplyConnectedPresence sets the presence right after a (re)connect.
// In human mode this explicitly announces "unavailable" so the account does not
// show up as online just because the bridge is running.
func (c *Client) ApplyConnectedPresence() {
	pm := c.presence
	if pm == nil || pm.mode == PresenceModeAlwaysOnline {
		if err := c.SetPresence("available"); err != nil {
			c.logger.Warnf("Presence: failed to set available: %v", err)
		} else {
			c.logger.Infof("✓ Presence set to available (mode: always_online)")
		}
		return
	}

	pm.mu.Lock()
	pm.gen++
	if pm.timer != nil {
		pm.timer.Stop()
		pm.timer = nil
	}
	pm.online = false
	pm.mu.Unlock()

	if err := c.SetPresence("unavailable"); err != nil {
		c.logger.Warnf("Presence: failed to set unavailable: %v", err)
	} else {
		c.logger.Infof("Presence: offline (mode: human)")
	}
}

// GoOnlineBriefly marks the account online for a short, randomised window.
// It is safe to call from any goroutine and on every outgoing action; repeated
// calls extend the window instead of stacking timers. No-op in always_online mode.
func (c *Client) GoOnlineBriefly() {
	pm := c.presence
	if pm == nil || pm.mode != PresenceModeHuman || !c.IsConnected() {
		return
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if !pm.online {
		if err := c.SetPresence("available"); err != nil {
			c.logger.Warnf("Presence: failed to go online: %v", err)
			return
		}
		pm.online = true
		c.logger.Infof("Presence: online (activity)")
	}

	pm.gen++
	gen := pm.gen
	delay := pm.linger()
	if pm.timer != nil {
		pm.timer.Stop()
	}
	pm.timer = time.AfterFunc(delay, func() { c.presenceLingerExpired(gen) })
}

// presenceLingerExpired drops the account back offline once no further activity
// happened during the linger window.
func (c *Client) presenceLingerExpired(gen uint64) {
	pm := c.presence
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if gen != pm.gen || !pm.online {
		return
	}

	if err := c.SetPresence("unavailable"); err != nil {
		c.logger.Warnf("Presence: failed to go offline: %v", err)
		return
	}
	pm.online = false
	pm.timer = nil
	c.logger.Infof("Presence: offline again (idle)")
}

// PresenceOnline reports whether the account is currently announced as online.
func (c *Client) PresenceOnline() bool {
	pm := c.presence
	if pm == nil {
		return true
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.online
}

// ResetPresenceState forgets the online flag after a disconnect, since the
// server-side presence is dropped together with the connection.
func (c *Client) ResetPresenceState() {
	pm := c.presence
	if pm == nil {
		return
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.gen++
	if pm.timer != nil {
		pm.timer.Stop()
		pm.timer = nil
	}
	pm.online = false
}
