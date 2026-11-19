package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/alexandrmotologa/framesieve/internal/stream"
	"github.com/alexandrmotologa/framesieve/internal/vision"
)

// DatasetEntry represents a single training / RAG observation item in JSONL.
type DatasetEntry struct {
	Timestamp        string                         `json:"timestamp"`
	SessionID        string                         `json:"session_id"`
	FrameID          int64                          `json:"frame_id"`
	SSIM             float64                        `json:"ssim"`
	Classification   *vision.SemanticClassification `json:"classification,omitempty"`
	ROI              *vision.BoundingBox            `json:"roi,omitempty"`
	TokensSent       int                            `json:"tokens_sent"`
	TokensSaved      int                            `json:"tokens_saved"`
	Base64DataURI    string                         `json:"base64_data_uri,omitempty"`
	PromptAnnotation string                         `json:"prompt_annotation"`
}

// DatasetRecorder writes visual keyframe bursts into a JSONL dataset file.
type DatasetRecorder struct {
	file      *os.File
	encoder   *json.Encoder
	sessionID string
	mu        sync.Mutex
	entryCount int64
}

// NewDatasetRecorder creates or appends to a JSONL dataset file.
func NewDatasetRecorder(filePath, sessionID string) (*DatasetRecorder, error) {
	if filePath == "" {
		return nil, nil
	}

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open dataset file %s: %w", filePath, err)
	}

	return &DatasetRecorder{
		file:      f,
		encoder:   json.NewEncoder(f),
		sessionID: sessionID,
	}, nil
}

// RecordEvent serializes a frame event with prompt annotations.
func (r *DatasetRecorder) RecordEvent(
	evt stream.FrameEvent,
	classification *vision.SemanticClassification,
) error {
	if r == nil || r.file == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	var desc string
	if classification != nil {
		desc = fmt.Sprintf("[%s (conf: %.2f)] %s", classification.Type, classification.Confidence, classification.Description)
	} else {
		desc = "[UI_STATE_CHANGE] Localized visual delta"
	}

	roiStr := "Full Frame"
	if evt.ROI != nil {
		roiStr = fmt.Sprintf("ROI: %dx%d at (%d, %d)", evt.ROI.Width, evt.ROI.Height, evt.ROI.X, evt.ROI.Y)
	}

	annotation := fmt.Sprintf("Observed %s. Spatial boundary: %s. Saved %d visual tokens.", desc, roiStr, evt.TokensSaved)

	frameID := int64(0)
	if evt.TriggerFrame != nil {
		frameID = evt.TriggerFrame.ID
	}

	entry := DatasetEntry{
		Timestamp:        evt.Timestamp.Format(time.RFC3339Nano),
		SessionID:        r.sessionID,
		FrameID:          frameID,
		SSIM:             evt.SSIM,
		Classification:   classification,
		ROI:              evt.ROI,
		TokensSent:       evt.TokensSent,
		TokensSaved:      evt.TokensSaved,
		Base64DataURI:    evt.Base64DataURI,
		PromptAnnotation: annotation,
	}

	r.entryCount++
	return r.encoder.Encode(entry)
}

// Close flushes and closes the dataset file.
func (r *DatasetRecorder) Close() error {
	if r == nil || r.file == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.file.Close()
}

// EntryCount returns the number of recorded training items.
func (r *DatasetRecorder) EntryCount() int64 {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.entryCount
}
