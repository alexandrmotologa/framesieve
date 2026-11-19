package vision

import (
	"testing"
)

func TestDetectMultiROIDisjointClusters(t *testing.T) {
	w, h := 640, 480
	lumA := make([]byte, w*h)
	lumB := make([]byte, w*h)

	for i := range lumA {
		lumA[i] = 40
		lumB[i] = 40
	}

	// Cluster 1: Top-Left (X: 50..120, Y: 40..100)
	for y := 40; y < 100; y++ {
		for x := 50; x < 120; x++ {
			lumB[y*w+x] = 200
		}
	}

	// Cluster 2: Bottom-Right (X: 450..550, Y: 350..420)
	for y := 350; y < 420; y++ {
		for x := 450; x < 550; x++ {
			lumB[y*w+x] = 220
		}
	}

	cluster := DetectMultiROI(lumA, lumB, w, h, 24, 64, 16, 28, 3)

	if len(cluster.Boxes) != 2 {
		t.Fatalf("expected 2 disjoint clusters, got %d", len(cluster.Boxes))
	}

	if !cluster.IsDisjoint {
		t.Fatal("expected IsDisjoint = true")
	}

	// Verify both boxes are aligned to 28
	for i, b := range cluster.Boxes {
		if b.Width%28 != 0 {
			t.Fatalf("cluster %d width %d not aligned to 28", i, b.Width)
		}
		if b.Height%28 != 0 {
			t.Fatalf("cluster %d height %d not aligned to 28", i, b.Height)
		}
	}

	// Test Packing
	rgb := make([]byte, w*h*3)
	composite, compBoxes := PackCompositeROIs(rgb, w, h, cluster.Boxes)
	if composite.Bounds().Dx() == 0 || composite.Bounds().Dy() == 0 {
		t.Fatal("composite image has 0 dimension")
	}
	if len(compBoxes) != 2 {
		t.Fatalf("expected 2 mapped composite boxes, got %d", len(compBoxes))
	}
}
