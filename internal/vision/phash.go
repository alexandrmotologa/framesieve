package vision

import (
	"math/bits"
)

// ComputeDHash calculates a 64-bit difference hash (dHash) from the luminance plane.
// It samples an 8x8 matrix of gradient transitions across 9 columns and 8 rows.
func ComputeDHash(lum []byte, width, height int) uint64 {
	if len(lum) < width*height || width < 9 || height < 8 {
		return 0
	}

	// 9 horizontal sample points and 8 vertical sample points
	xStep := width / 9
	yStep := height / 8
	var hash uint64

	bitIndex := 0
	for y := 0; y < 8; y++ {
		rowOffset := (y * yStep) * width
		for x := 0; x < 8; x++ {
			leftPix := lum[rowOffset+(x*xStep)]
			rightPix := lum[rowOffset+((x+1)*xStep)]

			if leftPix > rightPix {
				hash |= (1 << bitIndex)
			}
			bitIndex++
		}
	}

	return hash
}

// HammingDistance computes the bit difference between two 64-bit perceptual hashes.
// Uses CPU POPCNT instruction via math/bits.OnesCount64 (< 1ns).
func HammingDistance(h1, h2 uint64) int {
	return bits.OnesCount64(h1 ^ h2)
}
