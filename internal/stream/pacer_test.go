package stream

import (
	"testing"
	"time"
)

func TestTokenPacerEnforcement(t *testing.T) {
	// 6,000 tokens/min = 100 tokens/sec, min interval 50ms
	pacer := NewTokenPacer(6000, 50*time.Millisecond)

	// Burst 1: 500 tokens allowed
	if !pacer.Allow(500, 0.92) {
		t.Fatal("expected first burst to be allowed")
	}

	// Burst 2 immediately after: should be blocked by minBurstInterval
	if pacer.Allow(500, 0.92) {
		t.Fatal("expected immediate burst to be throttled by min interval")
	}

	// Burst 3: Emergency shock bypass (SSIM < 0.70)
	if !pacer.Allow(500, 0.65) {
		t.Fatal("expected emergency visual shock to bypass pacer interval")
	}
}
