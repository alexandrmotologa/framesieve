package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/framesieve/internal/config"
	"github.com/alexandrmotologa/framesieve/internal/stream"
	"github.com/alexandrmotologa/framesieve/web"
)

func TestServerEndpoints(t *testing.T) {
	cfg := config.DefaultConfig()
	engine := stream.NewEngine(cfg, 320, 240)
	distFS, _ := web.GetDistFS()

	srv := NewServer(cfg, engine, nil, distFS)

	// Test /api/telemetry
	req := httptest.NewRequest(http.MethodGet, "/api/telemetry", nil)
	rec := httptest.NewRecorder()
	srv.handleTelemetry(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /api/telemetry, got %d", rec.Code)
	}

	var snap stream.TelemetrySnapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snap); err != nil {
		t.Fatalf("failed to decode telemetry json: %v", err)
	}

	// Test static assets index.html
	reqIndex := httptest.NewRequest(http.MethodGet, "/", nil)
	recIndex := httptest.NewRecorder()
	srv.httpServer.Handler.ServeHTTP(recIndex, reqIndex)

	if recIndex.Code != http.StatusOK {
		t.Fatalf("expected 200 from static index.html, got %d", recIndex.Code)
	}
	if recIndex.Body.Len() == 0 {
		t.Fatal("empty response body from embedded index.html")
	}
}
