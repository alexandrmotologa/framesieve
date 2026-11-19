# framesieve API & CLI Reference

## 1. CLI Commands

### `framesieve stream`
Processes video streams in real-time, executing the perceptual sieve and optionally hosting the Studio console.

```bash
framesieve stream [options]
```

Options:
- `-input <path|url|synthetic>`: Video input source. Can be a local file (`.mp4`, `.mkv`), RTSP URL (`rtsp://...`), or `synthetic` (default: `synthetic`).
- `-config <path>`: Path to YAML configuration file.
- `-fps <int>`: Frame rate for ingestion (default: `30`).
- `-width <int>`: Frame width (default: `1280`).
- `-height <int>`: Frame height (default: `720`).
- `-port <int>`: Multimodal Studio port (default: `8686`).
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
Executes CPU SIMD latency benchmarks and memory footprint verification.

```bash
framesieve bench
```

Outputs:
- dHash latency per 1080p frame and throughput FPS.
- SSIM latency per 1080p and 720p frame.
- Dynamic ROI cluster extraction latency.
- Heap allocation and system memory usage.

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

```json
{
  "total_frames": 120,
  "passed_frames": 2,
  "dropped_frames": 118,
  "drop_ratio": 0.9833,
  "total_tokens_naive": 30345,
  "total_tokens_sent": 510,
  "tokens_saved": 29835,
  "saved_ratio": 0.9831,
  "est_dollars_saved": 0.0745,
  "avg_latency": 1140000,
  "last_event_time": "2026-11-19T22:15:00Z"
}
```

### `GET /api/events/sse`
Server-Sent Events (SSE) stream broadcasting real-time frame events:

```
event: frame_event
data: {
  "timestamp": "2026-11-19T22:15:01Z",
  "ssim": 0.9124,
  "roi": {"x": 200, "y": 150, "width": 280, "height": 280},
  "base64_data_uri": "data:image/jpeg;base64,...",
  "tokens_saved": 850,
  "tokens_sent": 255
}
```

### `POST /api/config`
Updates engine parameters dynamically:

```json
{
  "vision": {
    "ssim_threshold": 0.985,
    "phash_threshold": 2,
    "padding_margin": 32
  }
}
```
