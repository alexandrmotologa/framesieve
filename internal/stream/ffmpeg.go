package stream

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"github.com/alexandrmotologa/framesieve/internal/vision"
)

// FFmpegStreamConfig governs the ffmpeg ingestion process.
type FFmpegStreamConfig struct {
	InputURL string // File path, RTSP URL, or desktop capture device
	Width    int
	Height   int
	FPS      int
}

// IngestFFmpeg launches an ffmpeg process decoding any stream into raw RGB24 frames.
func IngestFFmpeg(
	ctx context.Context,
	cfg FFmpegStreamConfig,
	pool *vision.FramePool,
	out chan<- *vision.Frame,
) error {
	defer close(out)

	if cfg.Width <= 0 {
		cfg.Width = 1280
	}
	if cfg.Height <= 0 {
		cfg.Height = 720
	}
	if cfg.FPS <= 0 {
		cfg.FPS = 30
	}

	sizeArg := fmt.Sprintf("%dx%d", cfg.Width, cfg.Height)
	fpsArg := fmt.Sprintf("%d", cfg.FPS)

	args := []string{
		"-re", // Read input at native frame rate
		"-i", cfg.InputURL,
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-s", sizeArg,
		"-r", fpsArg,
		"-v", "error",
		"-",
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to open ffmpeg stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	frameSize := cfg.Width * cfg.Height * 3
	frameID := int64(1)

	readErrCh := make(chan error, 1)
	go func() {
		defer cmd.Wait()
		for {
			select {
			case <-ctx.Done():
				readErrCh <- ctx.Err()
				return
			default:
				frame := pool.Get()
				frame.ID = frameID
				frame.Timestamp = time.Now()

				_, err := io.ReadFull(stdout, frame.Data[:frameSize])
				if err != nil {
					pool.Put(frame)
					if err == io.EOF || err == io.ErrUnexpectedEOF {
						readErrCh <- nil
						return
					}
					readErrCh <- err
					return
				}

				frame.PopulateLuminance()

				select {
				case out <- frame:
					frameID++
				case <-ctx.Done():
					pool.Put(frame)
					readErrCh <- ctx.Err()
					return
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return ctx.Err()
	case err := <-readErrCh:
		return err
	}
}
