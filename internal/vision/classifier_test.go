package vision

import (
	"testing"
)

func TestSemanticClassifierTextInput(t *testing.T) {
	w, h := 640, 480
	lumA := make([]byte, w*h)
	lumB := make([]byte, w*h)

	box := BoundingBox{X: 100, Y: 100, Width: 300, Height: 120}

	// Simulate 3 lines of text: line at Y=10..22, Y=35..47, Y=60..72
	for line := 0; line < 3; line++ {
		startY := box.Y + 10 + line*25
		for y := startY; y < startY+12; y++ {
			for x := box.X + 10; x < box.X+240; x++ {
				lumB[y*w+x] = 200 // Pixel mutated
			}
		}
	}

	res := ClassifySemanticDelta(lumA, lumB, w, h, box, 24, Point{0, 0})
	if res.Type != DeltaTextInput {
		t.Fatalf("expected DeltaTextInput, got %s (conf: %f)", res.Type, res.Confidence)
	}
}

func TestSemanticClassifierModalPopup(t *testing.T) {
	w, h := 640, 480
	lumA := make([]byte, w*h)
	lumB := make([]byte, w*h)

	box := BoundingBox{X: 150, Y: 120, Width: 320, Height: 200}

	// Solid filled rectangular dialog
	for y := box.Y; y < box.Y+box.Height; y++ {
		for x := box.X; x < box.X+box.Width; x++ {
			lumB[y*w+x] = 220
		}
	}

	res := ClassifySemanticDelta(lumA, lumB, w, h, box, 24, Point{0, 0})
	if res.Type != DeltaModalPopup {
		t.Fatalf("expected DeltaModalPopup, got %s (conf: %f)", res.Type, res.Confidence)
	}
}

func TestAdaptiveNoiseTracker(t *testing.T) {
	tracker := NewAdaptiveNoiseTracker(0.2)
	w, h := 100, 100
	lumA := make([]byte, w*h)
	lumB := make([]byte, w*h)

	// Introduce subtle sensor noise (delta 3)
	for i := range lumA {
		lumA[i] = 50
		lumB[i] = 53
	}

	for i := 0; i < 5; i++ {
		tracker.Update(lumA, lumB, w, h)
	}

	effSSIM, effDelta := tracker.GetEffectiveThresholds(0.985, 24)
	if tracker.NoiseFloor() <= 0 {
		t.Fatal("expected positive noise floor")
	}
	t.Logf("Adaptive Noise Floor: %.2f, Effective SSIM: %.4f, Effective Delta: %d", tracker.NoiseFloor(), effSSIM, effDelta)
}
