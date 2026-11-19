//go:build windows

package stream

import (
	"context"
	"fmt"
	"syscall"
	"time"
	"unsafe"

	"github.com/alexandrmotologa/framesieve/internal/vision"
)

var (
	modUser32 = syscall.NewLazyDLL("user32.dll")
	modGdi32  = syscall.NewLazyDLL("gdi32.dll")

	procGetDC             = modUser32.NewProc("GetDC")
	procReleaseDC         = modUser32.NewProc("ReleaseDC")
	procGetSystemMetrics  = modUser32.NewProc("GetSystemMetrics")
	procCreateCompatibleDC = modGdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = modGdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject      = modGdi32.NewProc("SelectObject")
	procBitBlt            = modGdi32.NewProc("BitBlt")
	procGetDIBits         = modGdi32.NewProc("GetDIBits")
	procDeleteObject      = modGdi32.NewProc("DeleteObject")
	procDeleteDC          = modGdi32.NewProc("DeleteDC")
)

const (
	smCxScreen = 0
	smCyScreen = 1
	srccopy    = 0x00CC0020
	biRgb      = 0
)

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

// GetPrimaryScreenDimensions returns the native screen width and height on Windows.
func GetPrimaryScreenDimensions() (int, int) {
	w, _, _ := procGetSystemMetrics.Call(uintptr(smCxScreen))
	h, _, _ := procGetSystemMetrics.Call(uintptr(smCyScreen))
	return int(w), int(h)
}

// IngestNativeScreen captures the primary desktop monitor using Windows GDI.
func IngestNativeScreen(
	ctx context.Context,
	targetW, targetH, fps int,
	pool *vision.FramePool,
	out chan<- *vision.Frame,
) error {
	defer close(out)

	if fps <= 0 {
		fps = 30
	}
	frameDuration := time.Second / time.Duration(fps)

	screenW, screenH := GetPrimaryScreenDimensions()
	if screenW <= 0 || screenH <= 0 {
		return fmt.Errorf("failed to retrieve screen dimensions")
	}

	hdcScreen, _, _ := procGetDC.Call(0)
	if hdcScreen == 0 {
		return fmt.Errorf("failed to get desktop DC")
	}
	defer procReleaseDC.Call(0, hdcScreen)

	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return fmt.Errorf("failed to create compatible DC")
	}
	defer procDeleteDC.Call(hdcMem)

	hBitmap, _, _ := procCreateCompatibleBitmap.Call(hdcScreen, uintptr(screenW), uintptr(screenH))
	if hBitmap == 0 {
		return fmt.Errorf("failed to create compatible bitmap")
	}
	defer procDeleteObject.Call(hBitmap)

	oldObj, _, _ := procSelectObject.Call(hdcMem, hBitmap)
	defer procSelectObject.Call(hdcMem, oldObj)

	// Prepare BITMAPINFO for 32-bit BGRA extraction
	var bi bitmapInfoHeader
	bi.biSize = uint32(unsafe.Sizeof(bi))
	bi.biWidth = int32(screenW)
	bi.biHeight = -int32(screenH) // Negative for top-down bitmap
	bi.biPlanes = 1
	bi.biBitCount = 32
	bi.biCompression = biRgb

	bgraBuf := make([]byte, screenW*screenH*4)
	frameID := int64(1)
	ticker := time.NewTicker(frameDuration)
	defer ticker.Stop()

	scaleX := float64(screenW) / float64(targetW)
	scaleY := float64(screenH) / float64(targetH)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Blit screen to memory DC
			procBitBlt.Call(
				hdcMem, 0, 0, uintptr(screenW), uintptr(screenH),
				hdcScreen, 0, 0, srccopy,
			)

			// Extract pixels
			procGetDIBits.Call(
				hdcMem, hBitmap, 0, uintptr(screenH),
				uintptr(unsafe.Pointer(&bgraBuf[0])),
				uintptr(unsafe.Pointer(&bi)),
				0,
			)

			frame := pool.Get()
			frame.ID = frameID
			frame.Timestamp = time.Now()

			// Fast nearest-neighbor downsample to targetW x targetH and BGRA -> RGB24 conversion
			dst := frame.Data
			for y := 0; y < targetH; y++ {
				srcY := int(float64(y) * scaleY)
				if srcY >= screenH {
					srcY = screenH - 1
				}
				srcRow := srcY * screenW * 4
				dstRow := y * targetW * 3

				for x := 0; x < targetW; x++ {
					srcX := int(float64(x) * scaleX)
					if srcX >= screenW {
						srcX = screenW - 1
					}
					si := srcRow + srcX*4
					di := dstRow + x*3

					dst[di] = bgraBuf[si+2]   // R
					dst[di+1] = bgraBuf[si+1] // G
					dst[di+2] = bgraBuf[si]   // B
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
