package vision

import (
	"fmt"
)

// SemanticDeltaType defines the classified user interaction or visual change.
type SemanticDeltaType string

const (
	DeltaTextInput   SemanticDeltaType = "TEXT_INPUT"
	DeltaModalPopup  SemanticDeltaType = "MODAL_POPUP"
	DeltaButtonClick SemanticDeltaType = "BUTTON_CLICK"
	DeltaScroll      SemanticDeltaType = "SCROLL_TRANSLATION"
	DeltaMediaVideo  SemanticDeltaType = "MEDIA_PLAYBACK"
	DeltaGeneric     SemanticDeltaType = "UI_STATE_CHANGE"
)

// SemanticClassification holds the identified interaction type and confidence score.
type SemanticClassification struct {
	Type        SemanticDeltaType `json:"type"`
	Confidence  float64           `json:"confidence"`
	Description string            `json:"description"`
}

// ClassifySemanticDelta analyzes the pixel mutation geometry and projection profiles.
func ClassifySemanticDelta(
	lumA, lumB []byte,
	width, height int,
	box BoundingBox,
	deltaMin uint8,
	motion Point,
) SemanticClassification {
	if box.Width <= 0 || box.Height <= 0 {
		return SemanticClassification{Type: DeltaGeneric, Confidence: 0.5, Description: "Unknown empty delta"}
	}

	// Rule 1: Vertical scroll check
	if motion.X == 0 && (motion.Y > 4 || motion.Y < -4) && box.Height > height/3 {
		return SemanticClassification{
			Type:        DeltaScroll,
			Confidence:  0.92,
			Description: fmt.Sprintf("Vertical page translation by %d pixels", motion.Y),
		}
	}

	// Compute active pixels and 1D horizontal projection profile within the ROI
	horizProfile := make([]int, box.Height)
	totalActive := 0

	for y := 0; y < box.Height; y++ {
		rowOffset := (box.Y + y) * width
		rowActive := 0
		for x := 0; x < box.Width; x++ {
			idx := rowOffset + (box.X + x)
			if idx < len(lumA) && idx < len(lumB) {
				vA := lumA[idx]
				vB := lumB[idx]
				var diff uint8
				if vA > vB {
					diff = vA - vB
				} else {
					diff = vB - vA
				}
				if diff >= deltaMin {
					rowActive++
				}
			}
		}
		horizProfile[y] = rowActive
		totalActive += rowActive
	}

	totalArea := box.Width * box.Height
	if totalArea == 0 {
		return SemanticClassification{Type: DeltaGeneric, Confidence: 0.5, Description: "Zero area"}
	}

	fillDensity := float64(totalActive) / float64(totalArea)

	// Rule 2: Button / Icon Click Feedback
	if box.Width < 180 && box.Height < 70 && fillDensity > 0.15 {
		return SemanticClassification{
			Type:        DeltaButtonClick,
			Confidence:  0.88,
			Description: "Compact UI control or button interaction",
		}
	}

	// Rule 3: Text Input / Terminal Typing Detection
	// Analyzes alternating line periodicity (text line vs line spacing gap)
	lineGaps := 0
	linePeaks := 0
	inLine := false

	for y := 0; y < box.Height; y++ {
		rowDensity := float64(horizProfile[y]) / float64(box.Width)
		if rowDensity > 0.12 {
			if !inLine {
				linePeaks++
				inLine = true
			}
		} else {
			if inLine {
				lineGaps++
				inLine = false
			}
		}
	}

	if linePeaks >= 2 && lineGaps >= 1 && fillDensity < 0.45 {
		return SemanticClassification{
			Type:        DeltaTextInput,
			Confidence:  0.91,
			Description: fmt.Sprintf("Text input or code editing across %d lines", linePeaks),
		}
	}

	// Rule 4: Modal Dialog / Alert Pop-up
	if fillDensity > 0.48 && box.Width > 200 && box.Height > 80 {
		return SemanticClassification{
			Type:        DeltaModalPopup,
			Confidence:  0.89,
			Description: "Dense rectangular modal dialog or system alert",
		}
	}

	// Rule 5: Media / Video playback (high area, dense variable noise)
	if box.Area() > (width*height)/3 && fillDensity > 0.35 {
		return SemanticClassification{
			Type:        DeltaMediaVideo,
			Confidence:  0.82,
			Description: "Large area dynamic media or video stream playback",
		}
	}

	return SemanticClassification{
		Type:        DeltaGeneric,
		Confidence:  0.70,
		Description: "General localized UI state modification",
	}
}
