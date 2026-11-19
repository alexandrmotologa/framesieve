# framesieve API & CLI Reference

## 1. CLI Commands

### `framesieve stream`
Processes video streams in real-time, executing the perceptual sieve and optionally hosting the Studio console.

```bash
framesieve stream [options]
```

Options:
- `-input <path|url|screen|synthetic>`: Video input source. Can be a local file (`.mp4`, `.mkv`), RTSP URL (`rtsp://...`), `screen` (native Windows GDI desktop capture), or `synthetic` (default: `synthetic`).
- `-config <path>`: Path to YAML configuration file.
- `-fps <int>`: Frame rate for ingestion (default: `30`).
- `-width <int>`: Frame width (default: `1280`).
- `-height <int>`: Frame height (default: `720`).
- `-port <int>`: Multimodal Studio port (default: `8686`).
- `-record-dataset <path.jsonl>`: Optional path to export JSONL dataset for VLM fine-tuning.
- `-no-studio`: Disables the web studio server.
- `-db <path>`: SQLite database path for telemetry logging (default: `framesieve.db`).

---

### `framesieve proxy`
Starts the reverse proxy server intercepting OpenAI and Anthropic Vision requests.

```bash
framesieve proxy [options]
```

Options:
- `-listen <addr>`: Address to bind the proxy (default: `0.0.0.0:8088`).
- `-openai <url>`: Upstream OpenAI API base URL (default: `https://api.openai.com`).
- `-anthropic <url>`: Upstream Anthropic API base URL (default: `https://api.anthropic.com`).
- `-mock`: Enables mock upstream mode for local testing without valid API keys.
- `-db <path>`: SQLite database path for logging (default: `framesieve.db`).

---

### `framesieve bench`
Executes CPU SIMD latency benchmarks across all 6 core computer vision modules.

```bash
framesieve bench
```

Outputs:
1. Perceptual dHash latency (1080p).
2. Structural Similarity Index (SSIM 1080p).
3. Structural Similarity Index (SSIM 720p).
4. Dynamic ROI Cluster Extraction (1080p).
5. Multi-ROI Disjoint Cluster Extraction (1080p).
6. Semantic Delta Classification (Radon Projection Profiles).
7. Memory footprint (allocated heap and system memory).

---

### `framesieve studio`
Launches the standalone Multimodal Studio web console.

```bash
framesieve studio [-port 8686] [-host 0.0.0.0]
```

---

### `framesieve demo`
Launches an end-to-end interactive demonstration using the synthetic desktop generator and opens the Multimodal Studio on port 8686.

```bash
framesieve demo
```

---

## 2. HTTP REST & SSE Endpoints

When the Studio server is running (default port: `8686`):

### `GET /api/telemetry`
Returns current engine telemetry snapshot.

### `GET /api/events/sse`
Server-Sent Events (SSE) stream broadcasting real-time frame events:

```
event: frame_event
data: {
  "timestamp": "2026-11-19T22:15:01Z",
  "ssim": 0.9124,
  "roi": {"x": 200, "y": 150, "width": 280, "height": 280},
  "classification": {
    "type": "TEXT_INPUT",
    "confidence": 0.91,
    "description": "Text input or code editing across 3 lines"
  },
  "base64_data_uri": "data:image/jpeg;base64,...",
  "tokens_saved": 850,
  "tokens_sent": 255
}
```

### `POST /api/config`
Updates engine parameters dynamically at runtime.
