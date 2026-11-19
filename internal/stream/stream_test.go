package stream

import (
	"context"
	"testing"
	"time"

	"github.com/alexandrmotologa/framesieve/internal/config"
	"github.com/alexandrmotologa/framesieve/internal/vision"
)

func TestRingBuffer(t *testing.T) {
	pool := vision.NewFramePool(100, 100)
	rb := NewRingBuffer(3, pool)

	f1 := pool.Get()
	f1.ID = 1
	rb.Push(f1)

	f2 := pool.Get()
	f2.ID = 2
	rb.Push(f2)

	recent := rb.Recent(2)
	if len(recent) != 2 || recent[0].ID != 1 || recent[1].ID != 2 {
		t.Fatalf("unexpected recent frames order: %+v", recent)
	}

	// Push 2 more, overflowing capacity 3
	f3 := pool.Get()
	f3.ID = 3
	rb.Push(f3)

	f4 := pool.Get()
	f4.ID = 4
	rb.Push(f4)

	latest := rb.Latest()
	if latest == nil || latest.ID != 4 {
		t.Fatalf("expected latest frame ID 4, got %+v", latest)
	}

	all := rb.Recent(5)
	if len(all) != 3 {
		t.Fatalf("expected 3 frames after overflow, got %d", len(all))
	}
	if all[0].ID != 2 || all[1].ID != 3 || all[2].ID != 4 {
		t.Fatalf("unexpected chronological ring items: %v, %v, %v", all[0].ID, all[1].ID, all[2].ID)
	}

	rb.Clear()
}

func TestEngineSyntheticPipeline(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Vision.PHashThreshold = 2
	cfg.Vision.SSIMThreshold = 0.985

	width, height := 320, 240
	engine := NewEngine(cfg, width, height)

	eventsCh := engine.Subscribe()
	defer engine.Unsubscribe(eventsCh)

	synthCfg := SyntheticStreamConfig{
		Width:       width,
		Height:      height,
		FPS:         60,
		TotalFrames: 120, // 2 seconds
		EventPeriod: 30,  // Event every 30 frames
	}

	frameChan := make(chan *vision.Frame, 16)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_ = GenerateSyntheticFeed(ctx, synthCfg, engine.pool, frameChan)
	}()

	err := engine.ProcessStream(ctx, frameChan)
	if err != nil && err != context.Canceled {
		t.Fatalf("pipeline returned error: %v", err)
	}

	snap := engine.Snapshot()
	if snap.TotalFrames != 120 {
		t.Fatalf("expected 120 total frames processed, got %d", snap.TotalFrames)
	}

	// At least 85% of static frames should be dropped
	if snap.DropRatio < 0.85 {
		t.Fatalf("expected drop ratio >= 85%%, got %.2f%%", snap.DropRatio*100)
	}

	if snap.TokensSaved <= 0 {
		t.Fatalf("expected positive tokens saved, got %d", snap.TokensSaved)
	}

	t.Logf("Engine Results: Total Frames: %d, Dropped: %d (%.1f%%), Tokens Saved: %d (%.1f%%), Est USD Saved: $%.5f",
		snap.TotalFrames, snap.DroppedFrames, snap.DropRatio*100,
		snap.TokensSaved, snap.SavedRatio*100, snap.EstDollarsSaved,
	)
}
