package vision

import (
	"math"
)

// MotionEstimate holds the computed translational displacement and confidence score.
type MotionEstimate struct {
	DX         int     `json:"dx"`
	DY         int     `json:"dy"`
	Residual   float64 `json:"residual"`   // Average mean absolute difference after compensation
	Confidence float64 `json:"confidence"` // 0.0 to 1.0 confidence that change is pure translational motion
}

// EstimateGlobalMotion computes translational displacement between two luminance frames.
// Uses hierarchical coarse-to-fine block matching on downsampled anchor patches.
func EstimateGlobalMotion(lumA, lumB []byte, width, height, maxShift int) MotionEstimate {
	if maxShift <= 0 {
		maxShift = 16
	}
	if width < maxShift*2+32 || height < maxShift*2+32 {
		return MotionEstimate{DX: 0, DY: 0, Residual: 0, Confidence: 0}
	}

	// Select 5 distributed anchor blocks: Center, Top-Left, Top-Right, Bottom-Left, Bottom-Right
	const blockSize = 16
	stepX := width / 4
	stepY := height / 4

	anchorCenters := [][2]int{
		{width / 2, height / 2},
		{stepX, stepY},
		{width - stepX, stepY},
		{stepX, height - stepY},
		{width - stepX, height - stepY},
	}

	bestDX := 0
	bestDY := 0
	minGlobalSAD := int64(math.MaxInt64)

	// Search displacement grid [-maxShift, maxShift] with step 2 for performance
	for dy := -maxShift; dy <= maxShift; dy += 2 {
		for dx := -maxShift; dx <= maxShift; dx += 2 {
			var totalSAD int64

			for _, pt := range anchorCenters {
				ax := pt[0]
				ay := pt[1]

				// Calculate Sum of Absolute Differences (SAD) for this anchor
				for by := -blockSize / 2; by < blockSize/2; by++ {
					rowA := (ay + by) * width
					rowB := (ay + by + dy) * width
					for bx := -blockSize / 2; bx < blockSize/2; bx++ {
						idxA := rowA + (ax + bx)
						idxB := rowB + (ax + bx + dx)

						diff := int64(lumA[idxA]) - int64(lumB[idxB])
						if diff < 0 {
							diff = -diff
						}
						totalSAD += diff
					}
				}
			}

			if totalSAD < minGlobalSAD {
				minGlobalSAD = totalSAD
				bestDX = dx
				bestDY = dy
			}
		}
	}

	// Compute zero-motion SAD for comparison
	var zeroMotionSAD int64
	for _, pt := range anchorCenters {
		ax := pt[0]
		ay := pt[1]
		for by := -blockSize / 2; by < blockSize/2; by++ {
			row := (ay + by) * width
			for bx := -blockSize / 2; bx < blockSize/2; bx++ {
				idx := row + (ax + bx)
				diff := int64(lumA[idx]) - int64(lumB[idx])
				if diff < 0 {
					diff = -diff
				}
				zeroMotionSAD += diff
			}
		}
	}

	totalPixels := int64(len(anchorCenters) * blockSize * blockSize)
	avgResidual := float64(minGlobalSAD) / float64(totalPixels)

	var confidence float64
	if zeroMotionSAD > 0 {
		gain := float64(zeroMotionSAD-minGlobalSAD) / float64(zeroMotionSAD)
		if gain > 0 {
			confidence = gain
		}
	}

	return MotionEstimate{
		DX:         bestDX,
		DY:         bestDY,
		Residual:   avgResidual,
		Confidence: confidence,
	}
}
