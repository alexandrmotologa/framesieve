package stream

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alexandrmotologa/framesieve/internal/config"
	"github.com/alexandrmotologa/framesieve/internal/vision"
)

// TelemetrySnapshot captures instantaneous engine performance metrics.
type TelemetrySnapshot struct {
	TotalFrames      int64         `json:"total_frames"`
	PassedFrames     int64         `json:"passed_frames"`
	DroppedFrames    int64         `json:"dropped_frames"`
	DropRatio        float64       `json:"drop_ratio"`
	TotalTokensNaive int64         `json:"total_tokens_naive"`
	TotalTokensSent  int64         `json:"total_tokens_sent"`
	TokensSaved      int64         `json:"tokens_saved"`
	SavedRatio       float64       `json:"saved_ratio"`
	EstDollarsSaved  float64       `json:"est_dollars_saved"`
	AvgLatency       time.Duration `json:"avg_latency"`
	CurrentFPS       float64       `json:"current_fps"`
	LastEventTime    time.Time     `json:"last_event_time"`
}

// FrameEvent is dispatched whenever a significant semantic visual delta occurs.
type FrameEvent struct {
	Timestamp      time.Time           `json:"timestamp"`
	TriggerFrame   *vision.Frame       `json:"trigger_frame"`
	PreEventFrame  *vision.Frame       `json:"pre_event_frame,omitempty"`
	CroppedImage   []byte              `json:"-"`
	Base64DataURI  string              `json:"base64_data_uri,omitempty"`
	SSIM           float64             `json:"ssim"`
	ROI            *vision.BoundingBox `json:"roi,omitempty"`
	TokensSaved    int                 `json:"tokens_saved"`
	TokensSent     int                 `json:"tokens_sent"`
	FullTokensCost int                 `json:"full_tokens_cost"`
}

// Engine coordinates stream ingestion, vision sieving, ring buffer, and event dispatch.
type Engine struct {
	cfg        *config.SieveConfig
	pool       *vision.FramePool
	ring       *RingBuffer
	subscribers []chan FrameEvent
	subMu      sync.RWMutex

	// Atomic telemetry counters
	totalFrames      int64
	passedFrames     int64
	droppedFrames    int64
	totalTokensNaive int64
	totalTokensSent  int64
	tokensSaved      int64
	totalLatencyNs   int64

	lastRefFrame  *vision.Frame
	refMu         sync.Mutex
	lastEventTime time.Time
}

// NewEngine instantiates a streaming engine with allocated frame pools.
func NewEngine(cfg *config.SieveConfig, width, height int) *Engine {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	pool := vision.NewFramePool(width, height)
	ring := NewRingBuffer(cfg.Engine.RingBufferSize, pool)

	return &Engine{
		cfg:         cfg,
		pool:        pool,
		ring:        ring,
		subscribers: make([]chan FrameEvent, 0),
	}
}

// Pool returns the underlying memory pool for recyclable frames.
func (e *Engine) Pool() *vision.FramePool {
	return e.pool
}

// Subscribe returns a channel receiving high-signal visual events.
func (e *Engine) Subscribe() chan FrameEvent {
	e.subMu.Lock()
	defer e.subMu.Unlock()

	ch := make(chan FrameEvent, 32)
	e.subscribers = append(e.subscribers, ch)
	return ch
}

// Unsubscribe removes an event listener channel.
func (e *Engine) Unsubscribe(ch chan FrameEvent) {
	e.subMu.Lock()
	defer e.subMu.Unlock()

	for i, sub := range e.subscribers {
		if sub == ch {
			close(sub)
			e.subscribers = append(e.subscribers[:i], e.subscribers[i+1:]...)
			break
		}
	}
}

// ProcessStream runs the perceptual sieve loop on the incoming frame channel.
func (e *Engine) ProcessStream(ctx context.Context, in <-chan *vision.Frame) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case frame, ok := <-in:
			if !ok {
				return nil
			}
			e.processFrame(frame)
		}
	}
}

// processFrame executes the multi-stage filter on an individual frame.
func (e *Engine) processFrame(frame *vision.Frame) {
	start := time.Now()
	atomic.AddInt64(&e.totalFrames, 1)

	fullTokens := vision.CalculateVLMVisualTokens(frame.Width, frame.Height)
	atomic.AddInt64(&e.totalTokensNaive, int64(fullTokens))

	e.refMu.Lock()
	ref := e.lastRefFrame

	// First frame is always an anchor keyframe
	if ref == nil {
		frame.IsKey = true
		frame.SSIM = 1.0
		frame.PHash = vision.ComputeDHash(frame.Luminance, frame.Width, frame.Height)
		frame.Latency = time.Since(start)

		atomic.AddInt64(&e.passedFrames, 1)
		atomic.AddInt64(&e.totalTokensSent, int64(fullTokens))
		atomic.AddInt64(&e.totalLatencyNs, frame.Latency.Nanoseconds())

		// Store as reference
		refCopy := e.pool.Get()
		refCopy.ID = frame.ID
		refCopy.Width = frame.Width
		refCopy.Height = frame.Height
		copy(refCopy.Data, frame.Data)
		copy(refCopy.Luminance, frame.Luminance)
		refCopy.PHash = frame.PHash
		e.lastRefFrame = refCopy

		e.refMu.Unlock()
		e.ring.Push(frame)
		e.broadcastEvent(FrameEvent{
			Timestamp:      frame.Timestamp,
			TriggerFrame:   frame,
			FullTokensCost: fullTokens,
			TokensSent:     fullTokens,
			TokensSaved:    0,
			SSIM:           1.0,
		})
		return
	}

	// Stage 1: Fast pHash screening (< 0.1ms)
	frame.PHash = vision.ComputeDHash(frame.Luminance, frame.Width, frame.Height)
	dist := vision.HammingDistance(frame.PHash, ref.PHash)

	if dist <= e.cfg.Vision.PHashThreshold {
		// Frame is perceptually identical
		frame.Dropped = true
		frame.DropCode = "phash_static"
		frame.SSIM = 1.0
		frame.Latency = time.Since(start)

		atomic.AddInt64(&e.droppedFrames, 1)
		atomic.AddInt64(&e.tokensSaved, int64(fullTokens))
		atomic.AddInt64(&e.totalLatencyNs, frame.Latency.Nanoseconds())

		e.refMu.Unlock()
		e.pool.Put(frame)
		return
	}

	// Stage 2: Structural Similarity (SSIM) Gate (< 0.9ms)
	ssimVal := vision.ComputeSSIM(ref.Luminance, frame.Luminance, frame.Width, frame.Height, 8)
	frame.SSIM = ssimVal

	if ssimVal >= e.cfg.Vision.SSIMThreshold {
		frame.Dropped = true
		frame.DropCode = "ssim_static"
		frame.Latency = time.Since(start)

		atomic.AddInt64(&e.droppedFrames, 1)
		atomic.AddInt64(&e.tokensSaved, int64(fullTokens))
		atomic.AddInt64(&e.totalLatencyNs, frame.Latency.Nanoseconds())

		e.refMu.Unlock()
		e.pool.Put(frame)
		return
	}

	// Stage 3: Global Motion Estimation (Camera Jitter rejection)
	if e.cfg.Vision.EnableMotionCompensation {
		gme := vision.EstimateGlobalMotion(ref.Luminance, frame.Luminance, frame.Width, frame.Height, 8)
		frame.Motion = vision.Point{X: gme.DX, Y: gme.DY}

		// If motion is pure camera jitter within threshold with negligible residual
		if gme.Confidence > 0.85 && gme.Residual < e.cfg.Vision.MotionJitterThreshold {
			frame.Dropped = true
			frame.DropCode = "motion_jitter"
			frame.Latency = time.Since(start)

			atomic.AddInt64(&e.droppedFrames, 1)
			atomic.AddInt64(&e.tokensSaved, int64(fullTokens))
			atomic.AddInt64(&e.totalLatencyNs, frame.Latency.Nanoseconds())

			e.refMu.Unlock()
			e.pool.Put(frame)
			return
		}
	}

	// Stage 4: STATE CHANGE DETECTED - Dynamic ROI Extraction
	roi := vision.DetectROI(
		ref.Luminance, frame.Luminance,
		frame.Width, frame.Height,
		e.cfg.Vision.PixelDeltaMin,
		e.cfg.Vision.MinClusterArea,
		e.cfg.Vision.PaddingMargin,
		e.cfg.Vision.PatchGridAlign,
		e.cfg.Vision.MaxRoiExpansion,
	)

	frame.ROI = roi
	frame.IsKey = true

	var sentTokens int
	var croppedBytes []byte

	if roi != nil && (roi.Width < frame.Width || roi.Height < frame.Height) {
		// Crop localized patch
		croppedImg := vision.CropRGB(frame.Data, frame.Width, frame.Height, *roi)
		encoded, err := vision.EncodeImage(croppedImg, e.cfg.Proxy.OutputImageCodec, e.cfg.Proxy.ImageQuality)
		if err == nil {
			croppedBytes = encoded
			frame.Cropped = encoded
		}
		sentTokens = vision.CalculateVLMVisualTokens(roi.Width, roi.Height)
	} else {
		// Transmit full frame
		rgba := frame.ToRGBA()
		encoded, err := vision.EncodeImage(rgba, e.cfg.Proxy.OutputImageCodec, e.cfg.Proxy.ImageQuality)
		if err == nil {
			croppedBytes = encoded
			frame.Cropped = encoded
		}
		sentTokens = fullTokens
	}

	savedTokens := fullTokens - sentTokens
	if savedTokens < 0 {
		savedTokens = 0
	}

	frame.Latency = time.Since(start)
	atomic.AddInt64(&e.passedFrames, 1)
	atomic.AddInt64(&e.totalTokensSent, int64(sentTokens))
	atomic.AddInt64(&e.tokensSaved, int64(savedTokens))
	atomic.AddInt64(&e.totalLatencyNs, frame.Latency.Nanoseconds())

	// Update reference frame
	copy(ref.Data, frame.Data)
	copy(ref.Luminance, frame.Luminance)
	ref.PHash = frame.PHash
	ref.ID = frame.ID
	e.lastRefFrame = ref
	e.lastEventTime = frame.Timestamp
	e.refMu.Unlock()

	preEvent := e.ring.Latest()
	e.ring.Push(frame)

	var dataURI string
	if len(croppedBytes) > 0 {
		dataURI = vision.EncodeToBase64URI(croppedBytes, "image/jpeg")
	}

	e.broadcastEvent(FrameEvent{
		Timestamp:      frame.Timestamp,
		TriggerFrame:   frame,
		PreEventFrame:  preEvent,
		CroppedImage:   croppedBytes,
		Base64DataURI:  dataURI,
		SSIM:           ssimVal,
		ROI:            roi,
		TokensSaved:    savedTokens,
		TokensSent:     sentTokens,
		FullTokensCost: fullTokens,
	})
}

// broadcastEvent forwards event notifications to all subscribers.
func (e *Engine) broadcastEvent(evt FrameEvent) {
	e.subMu.RLock()
	defer e.subMu.RUnlock()

	for _, ch := range e.subscribers {
		select {
		case ch <- evt:
		default:
			// Non-blocking drop if consumer buffer is full
		}
	}
}

// Snapshot returns calculated telemetry statistics.
func (e *Engine) Snapshot() TelemetrySnapshot {
	total := atomic.LoadInt64(&e.totalFrames)
	passed := atomic.LoadInt64(&e.passedFrames)
	dropped := atomic.LoadInt64(&e.droppedFrames)
	naiveTokens := atomic.LoadInt64(&e.totalTokensNaive)
	sentTokens := atomic.LoadInt64(&e.totalTokensSent)
	savedTokens := atomic.LoadInt64(&e.tokensSaved)
	totalLatNs := atomic.LoadInt64(&e.totalLatencyNs)

	var dropRatio float64
	if total > 0 {
		dropRatio = float64(dropped) / float64(total)
	}

	var savedRatio float64
	if naiveTokens > 0 {
		savedRatio = float64(savedTokens) / float64(naiveTokens)
	}

	var avgLat time.Duration
	if total > 0 {
		avgLat = time.Duration(totalLatNs / total)
	}

	// Cost model: GPT-4o Vision input tokens @ $2.50 per 1,000,000 tokens ($0.0000025 per token)
	dollarsSaved := float64(savedTokens) * 0.0000025

	return TelemetrySnapshot{
		TotalFrames:      total,
		PassedFrames:     passed,
		DroppedFrames:    dropped,
		DropRatio:        dropRatio,
		TotalTokensNaive: naiveTokens,
		TotalTokensSent:  sentTokens,
		TokensSaved:      savedTokens,
		SavedRatio:       savedRatio,
		EstDollarsSaved:  dollarsSaved,
		AvgLatency:       avgLat,
		LastEventTime:    e.lastEventTime,
	}
}
