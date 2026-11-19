package vision

import (
	"testing"
)

func BenchmarkDHash1080p(b *testing.B) {
	w, h := 1920, 1080
	lum := make([]byte, w*h)
	for i := range lum {
		lum[i] = uint8(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ComputeDHash(lum, w, h)
	}
}

func BenchmarkSSIM1080p(b *testing.B) {
	w, h := 1920, 1080
	lumA := make([]byte, w*h)
	lumB := make([]byte, w*h)
	for i := range lumA {
		lumA[i] = uint8(i % 256)
		lumB[i] = uint8((i + 1) % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ComputeSSIM(lumA, lumB, w, h, 16)
	}
}

func BenchmarkDetectROI1080p(b *testing.B) {
	w, h := 1920, 1080
	lumA := make([]byte, w*h)
	lumB := make([]byte, w*h)
	for i := range lumA {
		lumA[i] = 100
		lumB[i] = 100
	}
	// Center delta
	for y := 400; y < 600; y++ {
		for x := 800; x < 1100; x++ {
			lumB[y*w+x] = 220
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DetectROI(lumA, lumB, w, h, 24, 64, 32, 28, 0.75)
	}
}
