package vision

import (
	"image"
)

// MultiROICluster holds detected disjoint bounding boxes.
type MultiROICluster struct {
	Boxes      []BoundingBox `json:"boxes"`
	TotalArea  int           `json:"total_area"`
	IsDisjoint bool          `json:"is_disjoint"`
}

// DetectMultiROI extracts up to maxClusters disjoint dynamic regions of interest.
func DetectMultiROI(
	lumA, lumB []byte,
	width, height int,
	deltaMin uint8,
	minArea int,
	padding int,
	patchAlign int,
	maxClusters int,
) MultiROICluster {
	if maxClusters <= 1 {
		single := DetectROI(lumA, lumB, width, height, deltaMin, minArea, padding, patchAlign, 0.75)
		if single == nil {
			return MultiROICluster{Boxes: nil, TotalArea: 0, IsDisjoint: false}
		}
		return MultiROICluster{Boxes: []BoundingBox{*single}, TotalArea: single.Area(), IsDisjoint: false}
	}

	const cellSize = 32
	gridCols := (width + cellSize - 1) / cellSize
	gridRows := (height + cellSize - 1) / cellSize

	grid := make([]bool, gridCols*gridRows)

	// Step 1: Mark active grid cells where pixel intensity delta exceeds threshold
	for gy := 0; gy < gridRows; gy++ {
		yStart := gy * cellSize
		yEnd := yStart + cellSize
		if yEnd > height {
			yEnd = height
		}

		for gx := 0; gx < gridCols; gx++ {
			xStart := gx * cellSize
			xEnd := xStart + cellSize
			if xEnd > width {
				xEnd = width
			}

			activeCount := 0
			// Sample 4 points in each cell for sub-millisecond screening
			for cy := yStart + 4; cy < yEnd; cy += 8 {
				rowOffset := cy * width
				for cx := xStart + 4; cx < xEnd; cx += 8 {
					idx := rowOffset + cx
					vA := lumA[idx]
					vB := lumB[idx]
					var diff uint8
					if vA > vB {
						diff = vA - vB
					} else {
						diff = vB - vA
					}
					if diff >= deltaMin {
						activeCount++
					}
				}
			}

			if activeCount >= 2 {
				grid[gy*gridCols+gx] = true
			}
		}
	}

	// Step 2: Connected Component Labeling on grid (8-connectivity BFS)
	visited := make([]bool, len(grid))
	var boxes []BoundingBox

	neighbors := [8][2]int{
		{-1, -1}, {0, -1}, {1, -1},
		{-1, 0}, {1, 0},
		{-1, 1}, {0, 1}, {1, 1},
	}

	for gy := 0; gy < gridRows; gy++ {
		for gx := 0; gx < gridCols; gx++ {
			idx := gy*gridCols + gx
			if !grid[idx] || visited[idx] {
				continue
			}

			// Flood-fill new component
			queue := [][2]int{{gx, gy}}
			visited[idx] = true

			minCellX, maxCellX := gx, gx
			minCellY, maxCellY := gy, gy
			cellCount := 0

			for len(queue) > 0 {
				curr := queue[0]
				queue = queue[1:]
				cellCount++

				cx, cy := curr[0], curr[1]
				if cx < minCellX {
					minCellX = cx
				}
				if cx > maxCellX {
					maxCellX = cx
				}
				if cy < minCellY {
					minCellY = cy
				}
				if cy > maxCellY {
					maxCellY = cy
				}

				for _, n := range neighbors {
					nx, ny := cx+n[0], cy+n[1]
					if nx >= 0 && nx < gridCols && ny >= 0 && ny < gridRows {
						nIdx := ny*gridCols + nx
						if grid[nIdx] && !visited[nIdx] {
							visited[nIdx] = true
							queue = append(queue, [2]int{nx, ny})
						}
					}
				}
			}

			// Convert grid cells back to pixel coordinates
			pxMin := minCellX * cellSize
			pyMin := minCellY * cellSize
			pxMax := (maxCellX + 1) * cellSize
			pyMax := (maxCellY + 1) * cellSize

			// Add padding and clamp
			pxMin -= padding
			pyMin -= padding
			pxMax += padding
			pyMax += padding

			if pxMin < 0 {
				pxMin = 0
			}
			if pyMin < 0 {
				pyMin = 0
			}
			if pxMax > width {
				pxMax = width
			}
			if pyMax > height {
				pyMax = height
			}

			boxW := pxMax - pxMin
			boxH := pyMax - pyMin

			// Align to VLM patch grid (e.g. 28px)
			if patchAlign > 1 {
				if rem := boxW % patchAlign; rem != 0 {
					boxW += patchAlign - rem
					if pxMin+boxW > width {
						pxMin = width - boxW
						if pxMin < 0 {
							pxMin = 0
							boxW = width
						}
					}
				}
				if rem := boxH % patchAlign; rem != 0 {
					boxH += patchAlign - rem
					if pyMin+boxH > height {
						pyMin = height - boxH
						if pyMin < 0 {
							pyMin = 0
							boxH = height
						}
					}
				}
			}

			if boxW > 0 && boxH > 0 && (boxW*boxH) >= minArea {
				boxes = append(boxes, BoundingBox{
					X:      pxMin,
					Y:      pyMin,
					Width:  boxW,
					Height: boxH,
				})
			}

			if len(boxes) >= maxClusters {
				break
			}
		}
		if len(boxes) >= maxClusters {
			break
		}
	}

	totalArea := 0
	for _, b := range boxes {
		totalArea += b.Area()
	}

	return MultiROICluster{
		Boxes:      boxes,
		TotalArea:  totalArea,
		IsDisjoint: len(boxes) > 1,
	}
}

// PackCompositeROIs packs multiple disjoint cropped ROIs into a single vertical composite image.
func PackCompositeROIs(src []byte, srcW, srcH int, boxes []BoundingBox) (*image.RGBA, []BoundingBox) {
	if len(boxes) == 0 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1)), nil
	}
	if len(boxes) == 1 {
		return CropRGB(src, srcW, srcH, boxes[0]), boxes
	}

	// Calculate composite dimensions (vertical stack)
	maxW := 0
	totalH := 0
	for _, b := range boxes {
		if b.Width > maxW {
			maxW = b.Width
		}
		totalH += b.Height
	}

	dst := image.NewRGBA(image.Rect(0, 0, maxW, totalH))
	compositeBoxes := make([]BoundingBox, len(boxes))

	currentY := 0
	for i, b := range boxes {
		subImg := CropRGB(src, srcW, srcH, b)
		sw := subImg.Bounds().Dx()
		sh := subImg.Bounds().Dy()

		for y := 0; y < sh; y++ {
			srcRow := y * subImg.Stride
			dstRow := (currentY + y) * dst.Stride
			copy(dst.Pix[dstRow:dstRow+sw*4], subImg.Pix[srcRow:srcRow+sw*4])
		}

		compositeBoxes[i] = BoundingBox{
			X:      0,
			Y:      currentY,
			Width:  sw,
			Height: sh,
		}
		currentY += sh
	}

	return dst, compositeBoxes
}
