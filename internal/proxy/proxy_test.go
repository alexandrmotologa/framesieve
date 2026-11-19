package proxy

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/framesieve/internal/config"
	"github.com/alexandrmotologa/framesieve/internal/vision"
)

func createTestImageURI(w, h int, c color.Color) string {
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

func TestProxyMockOpenAI(t *testing.T) {
	cfg := config.DefaultConfig()
	proxy := NewProxyServer(cfg, nil, true)

	uri1 := createTestImageURI(300, 300, color.RGBA{R: 50, G: 100, B: 150, A: 255})
	uri2 := createTestImageURI(300, 300, color.RGBA{R: 50, G: 100, B: 150, A: 255}) // Identical static frame

	reqBody := map[string]any{
		"model": "gpt-4o",
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": "Describe these video frames."},
					{"type": "image_url", "image_url": map[string]any{"url": uri1}},
					{"type": "image_url", "image_url": map[string]any{"url": uri2}},
				},
			},
		},
	}

	rawJSON, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(rawJSON))
	rec := httptest.NewRecorder()

	proxy.handleOpenAIChatCompletions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode mock response: %v", err)
	}

	choices, ok := resp["choices"].([]any)
	if !ok || len(choices) == 0 {
		t.Fatalf("missing choices in response: %+v", resp)
	}

	choice0 := choices[0].(map[string]any)
	msg := choice0["message"].(map[string]any)
	content := msg["content"].(string)

	t.Logf("Proxy Response: %s", content)
}
