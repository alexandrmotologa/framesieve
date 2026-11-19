package telemetry

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/alexandrmotologa/framesieve/internal/stream"
	"github.com/alexandrmotologa/framesieve/internal/vision"
)

func TestTelemetryStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_telemetry.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite store: %v", err)
	}
	defer store.Close()

	sessionID := "sess-001"
	if err := store.CreateSession(sessionID, "synthetic"); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Record a dropped frame metric
	evtDropped := stream.FrameEvent{
		Timestamp:   time.Now(),
		SSIM:        0.995,
		TokensSent:  0,
		TokensSaved: 1105,
		TriggerFrame: &vision.Frame{
			ID:       1,
			Dropped:  true,
			DropCode: "ssim_static",
		},
	}
	if err := store.RecordMetric(sessionID, evtDropped, 800*time.Microsecond); err != nil {
		t.Fatalf("failed to record dropped metric: %v", err)
	}

	// Record an active event with ROI
	evtActive := stream.FrameEvent{
		Timestamp:   time.Now(),
		SSIM:        0.910,
		TokensSent:  255,
		TokensSaved: 850,
		ROI: &vision.BoundingBox{
			X:      100,
			Y:      100,
			Width:  280,
			Height: 280,
		},
		TriggerFrame: &vision.Frame{
			ID:      2,
			Dropped: false,
		},
	}
	if err := store.RecordMetric(sessionID, evtActive, 1200*time.Microsecond); err != nil {
		t.Fatalf("failed to record active metric: %v", err)
	}

	stats, err := store.GetCumulativeStats()
	if err != nil {
		t.Fatalf("failed to query cumulative stats: %v", err)
	}

	if stats.TotalFrames != 2 {
		t.Fatalf("expected 2 total frames, got %d", stats.TotalFrames)
	}
	if stats.DroppedFrames != 1 {
		t.Fatalf("expected 1 dropped frame, got %d", stats.DroppedFrames)
	}
	if stats.TokensSaved != 1955 {
		t.Fatalf("expected 1955 tokens saved, got %d", stats.TokensSaved)
	}

	recent, err := store.GetRecentMetrics(10)
	if err != nil {
		t.Fatalf("failed to get recent metrics: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent metrics, got %d", len(recent))
	}
	if !recent[0].HasROI || recent[0].RoiW != 280 {
		t.Fatalf("unexpected recent metric ROI data: %+v", recent[0])
	}
}
