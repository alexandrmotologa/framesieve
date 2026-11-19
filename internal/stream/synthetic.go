package stream

import (
	"context"
	"time"

	"github.com/alexandrmotologa/framesieve/internal/vision"
)

// SyntheticStreamConfig controls the synthetic feed behavior.
type SyntheticStreamConfig struct {
	Width       int
	Height      int
	FPS         int
	TotalFrames int
	EventPeriod int // Every N frames, trigger a UI event
}

// DefaultSyntheticConfig provides a standard 1280x720 30fps desktop stream configuration.
func DefaultSyntheticConfig() SyntheticStreamConfig {
	return SyntheticStreamConfig{
		Width:       1280,
		Height:      720,
		FPS:         30,
		TotalFrames: 300, // 10 seconds of video
		EventPeriod: 30,  // Event once per second
	}
}

// GenerateSyntheticFeed produces a continuous stream of realistic desktop UI frames.
func GenerateSyntheticFeed(
	ctx context.Context,
	cfg SyntheticStreamConfig,
	pool *vision.FramePool,
	out chan<- *vision.Frame,
) error {
	defer close(out)

	if cfg.FPS <= 0 {
		cfg.FPS = 30
	}
	frameDuration := time.Second / time.Duration(cfg.FPS)

	// Pre-create baseline background RGB
	bg := make([]byte, cfg.Width*cfg.Height*3)
	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			idx := (y*cfg.Width + x) * 3
			if y < 36 {
				// Top dark menubar
				bg[idx] = 25
				bg[idx+1] = 28
				bg[idx+2] = 34
			} else {
				// Subtle desktop gradient
				bg[idx] = uint8(18 + (y * 12 / cfg.Height))
				bg[idx+1] = uint8(22 + (y * 14 / cfg.Height))
				bg[idx+2] = uint8(30 + (y * 18 / cfg.Height))
			}
		}
	}

	frameID := int64(1)
	ticker := time.NewTicker(frameDuration)
	defer ticker.Stop()

	termX := cfg.Width / 10
	termY := cfg.Height / 10
	termW := cfg.Width * 7 / 10
	termH := cfg.Height * 6 / 10

	textCursor := 0
	modalActive := false
	modalTimer := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if cfg.TotalFrames > 0 && int(frameID) > cfg.TotalFrames {
				return nil
			}

			frame := pool.Get()
			frame.ID = frameID
			frame.Timestamp = time.Now()
			copy(frame.Data, bg)

			// Draw Terminal Window
			for ty := termY; ty < termY+termH; ty++ {
				for tx := termX; tx < termX+termW; tx++ {
					idx := (ty*cfg.Width + tx) * 3
					if ty-termY < 24 {
						// Window titlebar
						frame.Data[idx] = 45
						frame.Data[idx+1] = 49
						frame.Data[idx+2] = 58
					} else {
						// Terminal body
						frame.Data[idx] = 15
						frame.Data[idx+1] = 17
						frame.Data[idx+2] = 22
					}
				}
			}

			// Periodic state transition: Typing in terminal
			if frameID%int64(cfg.EventPeriod) == 0 {
				textCursor = (textCursor + 1) % 15
			}

			// Render simulated text lines up to textCursor
			for line := 0; line <= textCursor; line++ {
				lineY := termY + 28 + line*16
				if lineY+8 >= termY+termH {
					break
				}
				maxLen := termW - 40
				if maxLen <= 0 {
					maxLen = 10
				}
				lineLen := 30 + ((line * 29) % maxLen)
				for lx := termX + 15; lx < termX+15+lineLen && lx < termX+termW-10 && lx < cfg.Width; lx++ {
					for h := 0; h < 6 && lineY+h < cfg.Height; h++ {
						idx := ((lineY + h) * cfg.Width + lx) * 3
						if idx+2 < len(frame.Data) {
							frame.Data[idx] = 160
							frame.Data[idx+1] = 220
							frame.Data[idx+2] = 170
						}
					}
				}
			}

			// Occasional Modal Notification appearing in bottom-right
			if frameID%90 == 60 {
				modalActive = true
				modalTimer = 25
			}
			if modalActive {
				mW := cfg.Width / 4
				mH := cfg.Height / 5
				if mW < 40 {
					mW = 40
				}
				if mH < 25 {
					mH = 25
				}
				mX := cfg.Width - mW - 10
				mY := cfg.Height - mH - 10
				for my := mY; my < mY+mH && my < cfg.Height; my++ {
					for mx := mX; mx < mX+mW && mx < cfg.Width; mx++ {
						idx := (my*cfg.Width + mx) * 3
						if idx+2 < len(frame.Data) {
							frame.Data[idx] = 50
							frame.Data[idx+1] = 70
							frame.Data[idx+2] = 140
						}
					}
				}
				modalTimer--
				if modalTimer <= 0 {
					modalActive = false
				}
			}

			frame.PopulateLuminance()

			select {
			case out <- frame:
				frameID++
			case <-ctx.Done():
				pool.Put(frame)
				return ctx.Err()
			}
		}
	}
}
