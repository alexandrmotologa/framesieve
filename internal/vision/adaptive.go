package vision

import (
	"sync"
)

// AdaptiveNoiseTracker dynamically models background sensor noise and compression variance.
type AdaptiveNoiseTracker struct {
	mu           sync.RWMutex
	emaMAD       float64 // Exponential Moving Average of Mean Absolute Difference
	alpha        float64 // Smoothing factor (default: 0.1)
	samplesCount int64
}

// NewAdaptiveNoiseTracker creates a noise estimator.
func NewAdaptiveNoiseTracker(alpha float64) *AdaptiveNoiseTracker {
	if alpha <= 0 || alpha >= 1 {
		alpha = 0.1
	}
	return &AdaptiveNoiseTracker{
		alpha: alpha,
	}
}

// Update evaluates static regions between reference and current frames.
func (t *AdaptiveNoiseTracker) Update(lumA, lumB []byte, width, height int) {
	if len(lumA) < width*height || len(lumB) < width*height || width <= 0 || height <= 0 {
		return
	}

	// Sample 64 distributed points to minimize CPU overhead (< 0.05 us)
	const sampleCount = 64
	stepX := width / 8
	stepY := height / 8

	var totalDiff int64
	for y := stepY / 2; y < height; y += stepY {
		row := y * width
		for x := stepX / 2; x < width; x += stepX {
			vA := int64(lumA[row+x])
			vB := int64(lumB[row+x])
			d := vA - vB
			if d < 0 {
				d = -d
			}
			totalDiff += d
		}
	}

	instantMAD := float64(totalDiff) / float64(sampleCount)

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.samplesCount == 0 {
		t.emaMAD = instantMAD
	} else {
		// Only update background noise if instant difference is small (under 15.0)
		if instantMAD < 15.0 {
			t.emaMAD = (1.0-t.alpha)*t.emaMAD + t.alpha*instantMAD
		}
	}
	t.samplesCount++
}

// GetEffectiveThresholds returns noise-adjusted SSIM and pixel delta thresholds.
func (t *AdaptiveNoiseTracker) GetEffectiveThresholds(baseSSIM float64, baseDelta uint8) (float64, uint8) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	effDelta := baseDelta
	noiseDelta := uint8(t.emaMAD * 2.2)
	if noiseDelta > effDelta {
		effDelta = noiseDelta
	}

	effSSIM := baseSSIM
	// For noisy sensors, slightly relax the SSIM cutoff to avoid false alarms
	if t.emaMAD > 2.0 {
		penalty := (t.emaMAD - 2.0) * 0.002
		effSSIM -= penalty
		if effSSIM < 0.940 {
			effSSIM = 0.940
		}
	}

	return effSSIM, effDelta
}

// NoiseFloor returns the estimated mean absolute noise level.
func (t *AdaptiveNoiseTracker) NoiseFloor() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.emaMAD
}
