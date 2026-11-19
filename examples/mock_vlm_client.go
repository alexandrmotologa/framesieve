package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"net/http"
	"time"

	"github.com/alexandrmotologa/framesieve/internal/vision"
)

func generateFrameDataURI(w, h int, c color.Color) string {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
	return vision.EncodeToBase64URI(buf.Bytes(), "image/jpeg")
}

func main() {
	proxyURL := "http://localhost:8088/v1/chat/completions"

	fmt.Println("Simulating Vision Agent sending 3 sequential video frames to framesieve proxy...")

	// Frame 1: Baseline screen
	f1 := generateFrameDataURI(800, 600, color.RGBA{R: 30, G: 40, B: 50, A: 255})
	// Frame 2: Identical static frame (should be filtered)
	f2 := generateFrameDataURI(800, 600, color.RGBA{R: 30, G: 40, B: 50, A: 255})
	// Frame 3: Localized state change
	img3 := image.NewRGBA(image.Rect(0, 0, 800, 600))
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			if x >= 200 && x < 400 && y >= 150 && y < 300 {
				img3.Set(x, y, color.RGBA{R: 220, G: 50, B: 50, A: 255}) // Modal alert
			} else {
				img3.Set(x, y, color.RGBA{R: 30, G: 40, B: 50, A: 255})
			}
		}
	}
	var buf3 bytes.Buffer
	_ = jpeg.Encode(&buf3, img3, &jpeg.Options{Quality: 85})
	f3 := vision.EncodeToBase64URI(buf3.Bytes(), "image/jpeg")

	payload := map[string]any{
		"model": "gpt-4o",
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": "Observe UI state transitions across sequential frames."},
					{"type": "image_url", "image_url": map[string]any{"url": f1}},
					{"type": "image_url", "image_url": map[string]any{"url": f2}},
					{"type": "image_url", "image_url": map[string]any{"url": f3}},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, proxyURL, bytes.NewReader(body))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Proxy connection failed (make sure framesieve proxy is running): %v\n", err)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	fmt.Printf("HTTP Status: %d\nResponse: %s\n", resp.StatusCode, string(respBody))
}
