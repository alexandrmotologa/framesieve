package vision

const (
	// Standard SSIM stability constants for 8-bit dynamic range (L = 255)
	// C1 = (0.01 * 255)^2 = 6.5025
	// C2 = (0.03 * 255)^2 = 58.5225
	c1 = 6.5025
	c2 = 58.5225
)

// ComputeSSIM computes the Structural Similarity Index between two luminance buffers.
// Uses an 8x8 block-based sliding approach with zero heap allocations.
// Returns a value in the range [-1.0, 1.0], where 1.0 indicates identical structure.
func ComputeSSIM(lumA, lumB []byte, width, height, stride int) float64 {
	if len(lumA) < width*height || len(lumB) < width*height || width < 8 || height < 8 {
		return 0.0
	}

	if stride <= 0 {
		stride = 8
	}

	var sumSSIM float64
	var blockCount int

	// Loop through non-overlapping or strided 8x8 blocks
	for y := 0; y <= height-8; y += stride {
		rowOffset := y * width
		for x := 0; x <= width-8; x += stride {
			var sumX, sumY, sumXX, sumYY, sumXY int64

			for by := 0; by < 8; by++ {
				idx := rowOffset + (by * width) + x
				for bx := 0; bx < 8; bx++ {
					vA := int64(lumA[idx+bx])
					vB := int64(lumB[idx+bx])

					sumX += vA
					sumY += vB
					sumXX += vA * vA
					sumYY += vB * vB
					sumXY += vA * vB
				}
			}

			// N = 64 pixels per block
			const n = 64.0
			muX := float64(sumX) / n
			muY := float64(sumY) / n

			varX := (float64(sumXX) / n) - (muX * muX)
			varY := (float64(sumYY) / n) - (muY * muY)
			covXY := (float64(sumXY) / n) - (muX * muY)

			num := (2.0*muX*muY + c1) * (2.0*covXY + c2)
			den := (muX*muX + muY*muY + c1) * (varX + varY + c2)

			if den > 0 {
				sumSSIM += num / den
			}
			blockCount++
		}
	}

	if blockCount == 0 {
		return 1.0
	}

	return sumSSIM / float64(blockCount)
}
