package telemetry

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alexandrmotologa/framesieve/internal/stream"
	"github.com/alexandrmotologa/framesieve/internal/vision"
)

func TestDatasetRecorder(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "dataset.jsonl")

	recorder, err := NewDatasetRecorder(outPath, "sess-test")
	if err != nil {
		t.Fatalf("failed to create recorder: %v", err)
	}

	evt := stream.FrameEvent{
		Timestamp:   time.Now(),
		SSIM:        0.915,
		TokensSent:  255,
		TokensSaved: 850,
		ROI: &vision.BoundingBox{
			X:      100,
			Y:      100,
			Width:  280,
			Height: 280,
		},
		Base64DataURI: "data:image/jpeg;base64,abc",
	}

	classification := &vision.SemanticClassification{
		Type:        vision.DeltaTextInput,
		Confidence:  0.94,
		Description: "Typing in terminal window",
	}

	if err := recorder.RecordEvent(evt, classification); err != nil {
		t.Fatalf("failed to record event: %v", err)
	}

	if recorder.EntryCount() != 1 {
		t.Fatalf("expected 1 entry, got %d", recorder.EntryCount())
	}
	recorder.Close()

	// Verify file content
	f, err := os.Open(outPath)
	if err != nil {
		t.Fatalf("failed to open output file: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		t.Fatal("expected at least one JSONL line")
	}
	line := scanner.Text()
	if len(line) == 0 {
		t.Fatal("empty line in dataset")
	}
}
