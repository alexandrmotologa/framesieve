package vision

import (
	"math/rand"
	"testing"
)

func TestDHashAndHamming(t *testing.T) {
	width, height := 64, 64
	lumA := make([]byte, width*height)
	lumB := make([]byte, width*height)

	// Create identical gradient
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			lumA[y*width+x] = uint8(x * 4)
			lumB[y*width+x] = uint8(x * 4)
		}
	}

	hA := ComputeDHash(lumA, width, height)
	hB := ComputeDHash(lumB, width, height)

	if dist := HammingDistance(hA, hB); dist != 0 {
		t.Fatalf("expected hamming distance 0 for identical images, got %d", dist)
	}

	// Invert gradient for B
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			lumB[y*width+x] = uint8((width - 1 - x) * 4)
		}
	}
	hBInv := ComputeDHash(lumB, width, height)
	if dist := HammingDistance(hA, hBInv); dist <= 10 {
		t.Fatalf("expected high hamming distance for inverted image, got %d", dist)
	}
}

func TestSSIMIdenticalAndDegraded(t *testing.T) {
	width, height := 128, 128
	lumA := make([]byte, width*height)
	lumB := make([]byte, width*height)

	// Populate with patterned data
	for i := 0; i < len(lumA); i++ {
		val := uint8((i*17 + 23) % 256)
		lumA[i] = val
		lumB[i] = val
	}

	ssimIdentical := ComputeSSIM(lumA, lumB, width, height, 8)
	if ssimIdentical < 0.999 {
		t.Fatalf("expected SSIM near 1.0 for identical frames, got %f", ssimIdentical)
	}

	// Add random noise to B
	r := rand.New(rand.NewSource(42))
	for i := 0; i < len(lumB); i++ {
		noise := int(lumB[i]) + r.Intn(100) - 50
		if noise < 0 {
			noise = 0
		}
		if noise > 255 {
			noise = 255
		}
		lumB[i] = uint8(noise)
	}

	ssimNoisy := ComputeSSIM(lumA, lumB, width, height, 8)
	if ssimNoisy >= ssimIdentical || ssimNoisy > 0.95 {
		t.Fatalf("expected degraded SSIM for noisy frames, got %f", ssimNoisy)
	}
}

func TestDetectROIAndPatchAlignment(t *testing.T) {
	width, height := 640, 480
	lumA := make([]byte, width*height)
	lumB := make([]byte, width*height)

	// Copy baseline
	for i := range lumA {
		lumA[i] = 50
		lumB[i] = 50
	}

	// Introduce a mutated rectangle in center: X: 200..300, Y: 150..220 (width 100, height 70)
	for y := 150; y < 220; y++ {
		for x := 200; x < 300; x++ {
			lumB[y*width+x] = 200 // Big delta
		}
	}

	padding := 10
	patchAlign := 28
	box := DetectROI(lumA, lumB, width, height, 24, 64, padding, patchAlign, 0.75)

	if box == nil {
		t.Fatal("expected non-nil ROI bounding box")
	}

	// Check alignment
	if box.Width%patchAlign != 0 {
		t.Fatalf("expected width to be multiple of %d, got %d", patchAlign, box.Width)
	}
	if box.Height%patchAlign != 0 {
		t.Fatalf("expected height to be multiple of %d, got %d", patchAlign, box.Height)
	}

	// Verify coordinates encompass the mutated region
	if box.X > 200 || box.X+box.Width < 300 {
		t.Fatalf("ROI bounding box X [%d..%d] misses active delta region [200..300]", box.X, box.X+box.Width)
	}
	if box.Y > 150 || box.Y+box.Height < 220 {
		t.Fatalf("ROI bounding box Y [%d..%d] misses active delta region [150..220]", box.Y, box.Y+box.Height)
	}
}

func TestCropAndTokenSavings(t *testing.T) {
	srcW, srcH := 1920, 1080
	srcRGB := make([]byte, srcW*srcH*3)

	// Fill background
	for i := 0; i < len(srcRGB); i += 3 {
		srcRGB[i] = 30
		srcRGB[i+1] = 60
		srcRGB[i+2] = 90
	}

	box := BoundingBox{X: 100, Y: 100, Width: 280, Height: 280}
	cropped := CropRGB(srcRGB, srcW, srcH, box)

	if cropped.Bounds().Dx() != 280 || cropped.Bounds().Dy() != 280 {
		t.Fatalf("expected cropped dimension 280x280, got %dx%d", cropped.Bounds().Dx(), cropped.Bounds().Dy())
	}

	fullTokens := CalculateVLMVisualTokens(srcW, srcH)
	cropTokens := CalculateVLMVisualTokens(box.Width, box.Height)

	if cropTokens >= fullTokens {
		t.Fatalf("expected cropped tokens (%d) < full frame tokens (%d)", cropTokens, fullTokens)
	}

	reductionPct := (1.0 - float64(cropTokens)/float64(fullTokens)) * 100
	if reductionPct < 50.0 {
		t.Fatalf("expected > 50%% token reduction, got %.1f%%", reductionPct)
	}
}

func TestFramePool(t *testing.T) {
	pool := NewFramePool(320, 240)
	f1 := pool.Get()
	if len(f1.Data) != 320*240*3 || len(f1.Luminance) != 320*240 {
		t.Fatalf("invalid buffer sizes from pool")
	}

	f1.Data[0] = 123
	pool.Put(f1)

	f2 := pool.Get()
	if f2 == nil {
		t.Fatal("expected recycled frame from pool")
	}
	pool.Put(f2)
}

func TestGlobalMotionEstimation(t *testing.T) {
	width, height := 200, 200
	lumA := make([]byte, width*height)
	lumB := make([]byte, width*height)

	// Create random-like texture
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			lumA[y*width+x] = uint8((x*13 + y*19) % 256)
		}
	}

	// Shift lumB by +4 in X and +2 in Y
	shiftX := 4
	shiftY := 2
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x - shiftX
			srcY := y - shiftY
			if srcX >= 0 && srcX < width && srcY >= 0 && srcY < height {
				lumB[y*width+x] = lumA[srcY*width+srcX]
			}
		}
	}

	est := EstimateGlobalMotion(lumA, lumB, width, height, 8)
	if est.DX != shiftX || est.DY != shiftY {
		t.Logf("estimated motion (%d, %d), expected (%d, %d) with confidence %f", est.DX, est.DY, shiftX, shiftY, est.Confidence)
	}
}
