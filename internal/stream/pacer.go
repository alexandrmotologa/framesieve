package stream

import (
	"sync"
	"time"
)

// TokenPacer controls output burst rate and enforces strict token budget limits.
type TokenPacer struct {
	mu                 sync.Mutex
	maxTokensPerMinute int
	minBurstInterval   time.Duration
	availableTokens    float64
	lastLeakTime       time.Time
	lastBurstTime      time.Time
}

// NewTokenPacer initializes a leaky-bucket pacer.
func NewTokenPacer(maxTokensPerMinute int, minInterval time.Duration) *TokenPacer {
	if maxTokensPerMinute <= 0 {
		maxTokensPerMinute = 100000 // Default 100k tokens / min
	}
	if minInterval <= 0 {
		minInterval = 200 * time.Millisecond // Default 5 bursts / sec max
	}

	return &TokenPacer{
		maxTokensPerMinute: maxTokensPerMinute,
		minBurstInterval:   minInterval,
		availableTokens:    float64(maxTokensPerMinute),
		lastLeakTime:       time.Now(),
		lastBurstTime:      time.Time{},
	}
}

// Allow checks whether a burst of tokenCost is allowed under the current budget.
// If ssim is below emergencyThreshold (0.70), it allows an immediate emergency bypass.
func (p *TokenPacer) Allow(tokenCost int, ssim float64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()

	// Emergency bypass for major visual shocks or urgent alerts
	if ssim < 0.70 {
		p.lastBurstTime = now
		return true
	}

	// Enforce minimum time interval between consecutive bursts
	if !p.lastBurstTime.IsZero() && now.Sub(p.lastBurstTime) < p.minBurstInterval {
		return false
	}

	// Leak tokens back into bucket: rate = maxTokensPerMinute / 60 seconds
	elapsedSec := now.Sub(p.lastLeakTime).Seconds()
	p.lastLeakTime = now

	leakRate := float64(p.maxTokensPerMinute) / 60.0
	p.availableTokens += elapsedSec * leakRate
	if p.availableTokens > float64(p.maxTokensPerMinute) {
		p.availableTokens = float64(p.maxTokensPerMinute)
	}

	cost := float64(tokenCost)
	if p.availableTokens >= cost {
		p.availableTokens -= cost
		p.lastBurstTime = now
		return true
	}

	return false
}
