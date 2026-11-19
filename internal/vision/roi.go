package vision

// DetectROI extracts the dynamic region of interest mutating between two luminance frames.
// Applies thresholding, bounding box clustering, context padding, and VLM patch grid alignment.
func DetectROI(
	lumA, lumB []byte,
	width, height int,
	deltaMin uint8,
	minArea int,
	padding int,
	patchAlign int,
	maxExpansion float64,
) *BoundingBox {
	if len(lumA) < width*height || len(lumB) < width*height || width <= 0 || height <= 0 {
		return nil
	}

	minX := width
	minY := height
	maxX := -1
	maxY := -1
	activePixels := 0

	// Step 2 downsampling for accelerated delta scan
	const step = 2

	for y := 0; y < height; y += step {
		rowOffset := y * width
		for x := 0; x < width; x += step {
			idx := rowOffset + x
			vA := lumA[idx]
			vB := lumB[idx]

			var diff uint8
			if vA > vB {
				diff = vA - vB
			} else {
				diff = vB - vA
			}

			if diff >= deltaMin {
				activePixels++
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}

	// Filter out sporadic isolated sensor noise
	if activePixels < minArea/(step*step) || maxX < minX || maxY < minY {
		return nil
	}

	// Expand by step margin
	maxX += step
	maxY += step

	// Add contextual padding margin
	minX -= padding
	minY -= padding
	maxX += padding
	maxY += padding

	// Clamp to image boundaries
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX > width {
		maxX = width
	}
	if maxY > height {
		maxY = height
	}

	boxW := maxX - minX
	boxH := maxY - minY

	if boxW <= 0 || boxH <= 0 {
		return nil
	}

	// Align to VLM patch grid (e.g. 28px multiples)
	if patchAlign > 1 {
		if rem := boxW % patchAlign; rem != 0 {
			boxW += patchAlign - rem
			if minX+boxW > width {
				minX = width - boxW
				if minX < 0 {
					minX = 0
					boxW = width
				}
			}
		}

		if rem := boxH % patchAlign; rem != 0 {
			boxH += patchAlign - rem
			if minY+boxH > height {
				minY = height - boxH
				if minY < 0 {
					minY = 0
					boxH = height
				}
			}
		}
	}

	// If mutated area covers the vast majority of the frame, return full frame
	totalArea := width * height
	boxArea := boxW * boxH
	if maxExpansion > 0 && float64(boxArea)/float64(totalArea) >= maxExpansion {
		return &BoundingBox{
			X:      0,
			Y:      0,
			Width:  width,
			Height: height,
		}
	}

	return &BoundingBox{
		X:      minX,
		Y:      minY,
		Width:  boxW,
		Height: boxH,
	}
}
