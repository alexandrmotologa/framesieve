//go:build !windows

package stream

import (
	"context"
	"fmt"

	"github.com/alexandrmotologa/framesieve/internal/vision"
)

// IngestNativeScreen stub for non-windows platforms.
func IngestNativeScreen(
	ctx context.Context,
	targetW, targetH, fps int,
	pool *vision.FramePool,
	out chan<- *vision.Frame,
) error {
	defer close(out)
	return fmt.Errorf("native desktop screen capture is currently implemented on Windows; use FFmpeg on Linux/macOS")
}
