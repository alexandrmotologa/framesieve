package vision

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
)

// CropRGB extracts a sub-rectangle from a raw RGB24 buffer without reallocating full images.
func CropRGB(src []byte, srcW, srcH int, box BoundingBox) *image.RGBA {
	// Clamp boundaries
	x := box.X
	y := box.Y
	w := box.Width
	h := box.Height

	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+w > srcW {
		w = srcW - x
	}
	if y+h > srcH {
		h = srcH - y
	}
	if w <= 0 || h <= 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	for row := 0; row < h; row++ {
		srcRowOffset := ((y + row) * srcW + x) * 3
		dstRowOffset := row * dst.Stride

		for col := 0; col < w; col++ {
			si := srcRowOffset + col*3
			di := dstRowOffset + col*4

			dst.Pix[di] = src[si]       // R
			dst.Pix[di+1] = src[si+1]   // G
			dst.Pix[di+2] = src[si+2]   // B
			dst.Pix[di+3] = 255         // A
		}
	}

	return dst
}

// EncodeImage encodes an image.Image to JPEG or PNG bytes.
func EncodeImage(img image.Image, format string, quality int) ([]byte, error) {
	var buf bytes.Buffer

	switch format {
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	default: // "jpeg" / "jpg"
		if quality <= 0 || quality > 100 {
			quality = 85
		}
		opts := &jpeg.Options{Quality: quality}
		if err := jpeg.Encode(&buf, img, opts); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// EncodeToBase64URI formats image bytes into a data URI RFC 2397 string for VLM payload.
func EncodeToBase64URI(data []byte, mimeType string) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
}

// CalculateVLMVisualTokens computes estimated token cost for an image of dimensions w x h.
// Uses the standard OpenAI high-detail patch formula:
// 1. Fits within 2048x2048 bounding box.
// 2. Shortest side scaled to 768px.
// 3. Tiled into 512x512 patches, each costing 170 tokens, plus 85 base tokens.
func CalculateVLMVisualTokens(w, h int) int {
	if w <= 0 || h <= 0 {
		return 0
	}

	// Base tokens
	const baseTokens = 85
	const tileTokens = 170
	const tileSize = 512

	// Scale down if larger than 2048x2048
	fw := float64(w)
	fh := float64(h)
	if fw > 2048 || fh > 2048 {
		scale := 2048.0 / math.Max(fw, fh)
		fw *= scale
		fh *= scale
	}

	// Scale shortest side to 768
	shortest := math.Min(fw, fh)
	if shortest > 768 {
		scale := 768.0 / shortest
		fw *= scale
		fh *= scale
	}

	tilesX := int(math.Ceil(fw / float64(tileSize)))
	tilesY := int(math.Ceil(fh / float64(tileSize)))
	if tilesX < 1 {
		tilesX = 1
	}
	if tilesY < 1 {
		tilesY = 1
	}

	return (tilesX * tilesY * tileTokens) + baseTokens
}
