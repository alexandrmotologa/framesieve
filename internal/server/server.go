package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/alexandrmotologa/framesieve/internal/config"
	"github.com/alexandrmotologa/framesieve/internal/stream"
	"github.com/alexandrmotologa/framesieve/internal/telemetry"
)

// Server coordinates the REST API, SSE live feed, and embedded studio assets.
type Server struct {
	cfg        *config.SieveConfig
	engine     *stream.Engine
	store      *telemetry.Store
	httpServer *http.Server
	clients    map[chan stream.FrameEvent]struct{}
	clientsMu  sync.RWMutex
	distFs     fs.FS
}

// NewServer initializes the HTTP studio server.
func NewServer(
	cfg *config.SieveConfig,
	engine *stream.Engine,
	store *telemetry.Store,
	distFs fs.FS,
) *Server {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	s := &Server{
		cfg:     cfg,
		engine:  engine,
		store:   store,
		clients: make(map[chan stream.FrameEvent]struct{}),
		distFs:  distFs,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/telemetry", s.handleTelemetry)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/events/sse", s.handleSSE)
	mux.HandleFunc("/api/config", s.handleConfig)

	// Web UI static assets
	if distFs != nil {
		fileServer := http.FileServer(http.FS(distFs))
		mux.Handle("/", fileServer)
	} else {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"service": "framesieve-multimodal-studio",
				"version": "1.0.0",
				"message": "React studio assets not yet embedded. Use /api/telemetry or /api/events/sse.",
			})
		})
	}

	addr := fmt.Sprintf("%s:%d", cfg.Studio.Host, cfg.Studio.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // Indefinite for SSE
	}

	// Subscribe to engine events and fan-out to SSE clients
	if engine != nil {
		eventCh := engine.Subscribe()
		go s.dispatchLoop(eventCh)
	}

	return s
}

// Start runs the HTTP server.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Close terminates the server.
func (s *Server) Close() error {
	return s.httpServer.Close()
}

func (s *Server) dispatchLoop(eventCh chan stream.FrameEvent) {
	for evt := range eventCh {
		s.clientsMu.RLock()
		for ch := range s.clients {
			select {
			case ch <- evt:
			default:
				// Skip if client buffer is blocked
			}
		}
		s.clientsMu.RUnlock()
	}
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var snap stream.TelemetrySnapshot
	if s.engine != nil {
		snap = s.engine.Snapshot()
	}

	_ = json.NewEncoder(w).Encode(snap)
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if s.store == nil {
		_ = json.NewEncoder(w).Encode([]any{})
		return
	}

	metrics, err := s.store.GetRecentMetrics(60)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(metrics)
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientCh := make(chan stream.FrameEvent, 32)
	s.clientsMu.Lock()
	s.clients[clientCh] = struct{}{}
	s.clientsMu.Unlock()

	defer func() {
		s.clientsMu.Lock()
		delete(s.clients, clientCh)
		s.clientsMu.Unlock()
		close(clientCh)
	}()

	notify := r.Context().Done()

	for {
		select {
		case <-notify:
			return
		case evt, ok := <-clientCh:
			if !ok {
				return
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: frame_event\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		_ = json.NewEncoder(w).Encode(s.cfg)
		return
	}

	if r.Method == http.MethodPost {
		var incoming config.SieveConfig
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		// Update dynamic thresholds
		if incoming.Vision.SSIMThreshold > 0 {
			s.cfg.Vision.SSIMThreshold = incoming.Vision.SSIMThreshold
		}
		if incoming.Vision.PHashThreshold >= 0 {
			s.cfg.Vision.PHashThreshold = incoming.Vision.PHashThreshold
		}
		if incoming.Vision.PaddingMargin >= 0 {
			s.cfg.Vision.PaddingMargin = incoming.Vision.PaddingMargin
		}
		_ = json.NewEncoder(w).Encode(s.cfg)
		return
	}

	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
