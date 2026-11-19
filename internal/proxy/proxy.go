package proxy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/framesieve/internal/config"
	"github.com/alexandrmotologa/framesieve/internal/telemetry"
	"github.com/alexandrmotologa/framesieve/internal/vision"
)

// ProxyServer acts as a transparent reverse proxy for OpenAI and Anthropic Vision APIs.
type ProxyServer struct {
	cfg       *config.SieveConfig
	store     *telemetry.Store
	server    *http.Server
	lastLum   []byte
	lastWidth int
	lastHeight int
	lastPHash uint64
	mu        sync.Mutex
	sessionID string
	isMock    bool
}

// NewProxyServer initializes a reverse proxy server.
func NewProxyServer(cfg *config.SieveConfig, store *telemetry.Store, isMock bool) *ProxyServer {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	sessionID := fmt.Sprintf("proxy-%d", time.Now().UnixNano())
	if store != nil {
		_ = store.CreateSession(sessionID, "vlm-reverse-proxy")
	}

	p := &ProxyServer{
		cfg:       cfg,
		store:     store,
		sessionID: sessionID,
		isMock:    isMock,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", p.handleOpenAIChatCompletions)
	mux.HandleFunc("/v1/messages", p.handleAnthropicMessages)
	mux.HandleFunc("/healthz", p.handleHealthz)

	p.server = &http.Server{
		Addr:         cfg.Proxy.ListenAddr,
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	return p
}

// Start launches the HTTP proxy server.
func (p *ProxyServer) Start() error {
	return p.server.ListenAndServe()
}

// Close gracefully stops the server.
func (p *ProxyServer) Close() error {
	return p.server.Close()
}

// Addr returns the configured listening address.
func (p *ProxyServer) Addr() string {
	return p.cfg.Proxy.ListenAddr
}

func (p *ProxyServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "healthy",
		"service":    "framesieve-vlm-proxy",
		"session_id": p.sessionID,
		"timestamp":  time.Now(),
	})
}

// handleOpenAIChatCompletions intercepts and optimizes OpenAI vision payloads.
func (p *ProxyServer) handleOpenAIChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Optimize image contents inside messages
	optimizedBody, tokensSaved, tokensSent := p.optimizeOpenAIBody(payload)

	// If mock mode is active, return simulated completion immediately
	if p.isMock {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"id":      fmt.Sprintf("chatcmpl-framesieve-%d", time.Now().Unix()),
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   payload["model"],
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": fmt.Sprintf("[framesieve-mock] Processed visual payload: %d tokens sent, %d redundant tokens saved.", tokensSent, tokensSaved),
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     tokensSent,
				"completion_tokens": 24,
				"total_tokens":      tokensSent + 24,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Forward to real upstream OpenAI
	targetURL, err := url.Parse(p.cfg.Proxy.UpstreamOpenAI)
	if err != nil {
		http.Error(w, "Invalid upstream OpenAI URL", http.StatusInternalServerError)
		return
	}

	proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL.String()+"/v1/chat/completions", bytes.NewReader(optimizedBody))
	if err != nil {
		http.Error(w, "Failed to create upstream request", http.StatusInternalServerError)
		return
	}

	for k, vv := range r.Header {
		for _, v := range vv {
			proxyReq.Header.Add(k, v)
		}
	}
	proxyReq.Header.Set("Content-Length", fmt.Sprintf("%d", len(optimizedBody)))

	client := &http.Client{Timeout: 90 * time.Second}
	upstreamResp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Upstream request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer upstreamResp.Body.Close()

	for k, vv := range upstreamResp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(upstreamResp.StatusCode)
	_, _ = io.Copy(w, upstreamResp.Body)
}

// handleAnthropicMessages intercepts and optimizes Anthropic vision payloads.
func (p *ProxyServer) handleAnthropicMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Optimize Anthropic image content arrays
	optimizedBody, tokensSaved, tokensSent := p.optimizeAnthropicBody(payload)

	if p.isMock {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"id":      fmt.Sprintf("msg-framesieve-%d", time.Now().Unix()),
			"type":    "message",
			"role":    "assistant",
			"model":   payload["model"],
			"content": []map[string]any{{"type": "text", "text": fmt.Sprintf("[framesieve-mock] Anthropic request processed: %d tokens sent, %d tokens saved.", tokensSent, tokensSaved)}},
			"usage":   map[string]any{"input_tokens": tokensSent, "output_tokens": 20},
		}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	targetURL, err := url.Parse(p.cfg.Proxy.UpstreamAnthropic)
	if err != nil {
		http.Error(w, "Invalid upstream Anthropic URL", http.StatusInternalServerError)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	r.Body = io.NopCloser(bytes.NewReader(optimizedBody))
	r.ContentLength = int64(len(optimizedBody))
	proxy.ServeHTTP(w, r)
}

func (p *ProxyServer) optimizeOpenAIBody(payload map[string]any) ([]byte, int, int) {
	messages, ok := payload["messages"].([]any)
	if !ok {
		b, _ := json.Marshal(payload)
		return b, 0, 0
	}

	totalSaved := 0
	totalSent := 0

	for _, msgItem := range messages {
		msg, ok := msgItem.(map[string]any)
		if !ok {
			continue
		}
		contentList, ok := msg["content"].([]any)
		if !ok {
			continue
		}

		newContentList := make([]any, 0, len(contentList))

		for _, item := range contentList {
			part, ok := item.(map[string]any)
			if !ok {
				newContentList = append(newContentList, item)
				continue
			}

			partType, _ := part["type"].(string)
			if partType != "image_url" {
				newContentList = append(newContentList, part)
				continue
			}

			imgObj, _ := part["image_url"].(map[string]any)
			urlStr, _ := imgObj["url"].(string)

			if !strings.HasPrefix(urlStr, "data:image/") {
				newContentList = append(newContentList, part)
				continue
			}

			// Parse Data URI
			rawBytes, _, err := parseDataURI(urlStr)
			if err != nil {
				newContentList = append(newContentList, part)
				continue
			}

			img, _, err := image.Decode(bytes.NewReader(rawBytes))
			if err != nil {
				newContentList = append(newContentList, part)
				continue
			}

			bounds := img.Bounds()
			w := bounds.Dx()
			h := bounds.Dy()
			fullTokens := vision.CalculateVLMVisualTokens(w, h)

			lum := extractLuminance(img)
			dhash := vision.ComputeDHash(lum, w, h)

			p.mu.Lock()
			if p.lastLum == nil || p.lastWidth != w || p.lastHeight != h {
				p.lastLum = lum
				p.lastWidth = w
				p.lastHeight = h
				p.lastPHash = dhash
				p.mu.Unlock()

				totalSent += fullTokens
				newContentList = append(newContentList, part)
				continue
			}

			dist := vision.HammingDistance(dhash, p.lastPHash)
			if dist <= p.cfg.Vision.PHashThreshold {
				// Static frame: omit or replace with reference indicator
				p.mu.Unlock()
				totalSaved += fullTokens
				newContentList = append(newContentList, map[string]any{
					"type": "text",
					"text": "[framesieve: Identical static frame omitted to preserve tokens]",
				})
				continue
			}

			ssimVal := vision.ComputeSSIM(p.lastLum, lum, w, h, 8)
			if ssimVal >= p.cfg.Vision.SSIMThreshold {
				p.mu.Unlock()
				totalSaved += fullTokens
				newContentList = append(newContentList, map[string]any{
					"type": "text",
					"text": fmt.Sprintf("[framesieve: Perceptually redundant frame omitted, SSIM=%.4f]", ssimVal),
				})
				continue
			}

			// Dynamic ROI detected
			roi := vision.DetectROI(
				p.lastLum, lum, w, h,
				p.cfg.Vision.PixelDeltaMin,
				p.cfg.Vision.MinClusterArea,
				p.cfg.Vision.PaddingMargin,
				p.cfg.Vision.PatchGridAlign,
				p.cfg.Vision.MaxRoiExpansion,
			)

			// Update reference
			p.lastLum = lum
			p.lastPHash = dhash
			p.mu.Unlock()

			if roi != nil && (roi.Width < w || roi.Height < h) {
				cropped := cropImage(img, *roi)
				encoded, err := vision.EncodeImage(cropped, "jpeg", 85)
				if err == nil {
					cropTokens := vision.CalculateVLMVisualTokens(roi.Width, roi.Height)
					totalSent += cropTokens
					totalSaved += (fullTokens - cropTokens)

					newContentList = append(newContentList, map[string]any{
						"type": "text",
						"text": fmt.Sprintf("[framesieve: Dynamic ROI active delta at X=%d, Y=%d, W=%d, H=%d]", roi.X, roi.Y, roi.Width, roi.Height),
					})
					newContentList = append(newContentList, map[string]any{
						"type": "image_url",
						"image_url": map[string]any{
							"url": vision.EncodeToBase64URI(encoded, "image/jpeg"),
						},
					})
					continue
				}
			}

			totalSent += fullTokens
			newContentList = append(newContentList, part)
		}

		msg["content"] = newContentList
	}

	payload["messages"] = messages
	outBytes, _ := json.Marshal(payload)
	return outBytes, totalSaved, totalSent
}

func (p *ProxyServer) optimizeAnthropicBody(payload map[string]any) ([]byte, int, int) {
	// Re-marshal for now with tracking
	b, _ := json.Marshal(payload)
	return b, 0, 0
}

func parseDataURI(dataURI string) ([]byte, string, error) {
	parts := strings.SplitN(dataURI, ",", 2)
	if len(parts) != 2 {
		return nil, "", fmt.Errorf("invalid data uri")
	}

	meta := parts[0]
	data := parts[1]

	var mimeType string
	if strings.HasPrefix(meta, "data:") {
		meta = strings.TrimPrefix(meta, "data:")
		mimeParts := strings.Split(meta, ";")
		mimeType = mimeParts[0]
	}

	decoded, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, "", err
	}

	return decoded, mimeType, nil
}

func extractLuminance(img image.Image) []byte {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	lum := make([]byte, w*h)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, _ := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			// RGBA() returns alpha-premultiplied values in [0, 65535]
			r8 := r >> 8
			g8 := g >> 8
			b8 := b >> 8
			lum[y*w+x] = uint8((19595*r8 + 38470*g8 + 7471*b8) >> 16)
		}
	}
	return lum
}

func cropImage(img image.Image, box vision.BoundingBox) *image.RGBA {
	bounds := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, box.Width, box.Height))

	for y := 0; y < box.Height; y++ {
		for x := 0; x < box.Width; x++ {
			srcX := bounds.Min.X + box.X + x
			srcY := bounds.Min.Y + box.Y + y
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}
	return dst
}
