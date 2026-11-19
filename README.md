# framesieve

Real-time perceptual video stream compressor and dynamic Region-of-Interest (ROI) extractor for Vision-Language Models.

`framesieve` evaluates video streams (desktop screencasts, webcam feeds, and CCTV streams) on CPU, filtering out redundant frames and extracting active visual deltas before sending payloads to upstream Vision-Language Models (GPT-4o Vision, Gemini 1.5 Pro, Claude Sonnet).

---

## Technical Specifications

- **Processing Latency**: Under 1.5ms per 1080p frame on CPU.
- **Memory Footprint**: Under 15MB RAM steady state via preallocated buffer pools.
- **Token Reduction**: Discards 90% to 95% of static frames, cutting visual token costs by up to 98%.
- **VLM Patch Grid Alignment**: Automatically snaps dynamic crop boundaries to 28px patch multiples, eliminating fractional token waste.
- **Multi-ROI Spatial Packing**: Disjoint cluster extraction isolating multiple distant UI mutations without capturing the whole screen.
- **Adaptive Noise Floor**: Dynamic SSIM and delta threshold calibration using Exponential Moving Averages (EMA) to prevent false positives from sensor grain or MPEG macroblocking.
- **Semantic Delta Classification**: Real-time classification of visual deltas (`TEXT_INPUT`, `MODAL_POPUP`, `BUTTON_CLICK`, `SCROLL_TRANSLATION`, `MEDIA_PLAYBACK`) using 1D Radon projection profiles.
- **Native Desktop Capture**: Direct zero-dependency screen capture on Windows via GDI32.
- **Dataset Recorder**: Direct export of high-signal visual events to JSONL for multimodal SFT fine-tuning and Vision RAG.
- **Embedded Visual Studio**: Bundles a React 19 studio inside the single Go executable with Normal, Delta Heatmap, and Token Grid viewing modes.

---

## Architecture Pipeline

```
       +-------------------------------------------------------------+
       |             Continuous Video Stream (30 / 60 FPS)           |
       |  (Native Screen Capture, Camera Feed, RTSP / WebRTC, VOD)   |
       +-------------------------------------------------------------+
                                      |
                      Streaming RGB24 / YUV Video Frames
                                      v
       +-------------------------------------------------------------+
       |                   framesieve Pipeline Engine                |
       |                                                             |
       |  [Stage 1] 64-bit dHash Screen (< 0.12 us)                  |
       |            Hamming Distance <= 2 -> Drop Static Frame       |
       |                                                             |
       |  [Stage 2] 8x8 Block SSIM Luminance Gate (< 0.9 ms)         |
       |            + Adaptive Noise Floor Calibration (EMA)         |
       |            SSIM >= EffSSIM -> Drop Static Frame             |
       |                                                             |
       |  [Stage 3] Global Motion Estimation (GME)                   |
       |            Camera Jitter Filter (< 1.5 px shift)            |
       |                                                             |
       |  [Stage 4] Dynamic ROI / Multi-ROI Disjoint Clustering      |
       |            Semantic Delta Classifier (Radon Projections)    |
       |            Patch-Grid Alignment (28px tiles)                |
       |            Token Budget Leaky Bucket Pacing                 |
       +-------------------------------------------------------------+
                                      |
                       Compact Visual Event Burst
                                      v
       +-------------------------------------------------------------+
       |                Upstream Vision-Language Model               |
       |            (GPT-4o Vision, Gemini 1.5 Pro, Claude)           |
       +-------------------------------------------------------------+
```

---

## Installation

### Prerequisites

- Go 1.22+ (tested on Go 1.27)
- Node.js 18+ (for building the React 19 studio)
- FFmpeg (optional, required only for video files, RTSP streams, and hardware devices; not required for native desktop screen capture)

### Build from Source

```bash
# Clone the repository
git clone https://github.com/alexandrmotologa/framesieve.git
cd framesieve

# Build embedded web assets and Go binary
make all

# Or build the binary directly:
go build -o framesieve ./cmd/framesieve
```

---

## Quickstart

### 1. Instant Interactive Demo

Run the built-in synthetic desktop stream and launch the Multimodal Studio on port 8686:

```bash
./framesieve demo
```

Open `http://localhost:8686` in your browser to inspect live SSIM curves, active ROI bounding boxes, semantic classification tags, and real-time token savings.

### 2. Stream Ingestion & Native Desktop Capture

```bash
# Capture primary display directly on Windows (no FFmpeg required)
./framesieve stream -input screen -fps 30

# Ingest local MP4 recording
./framesieve stream -input recording.mp4 -fps 30

# Ingest live RTSP camera stream
./framesieve stream -input rtsp://192.168.1.100:554/live -fps 30

# Ingest and simultaneously record fine-tuning dataset in JSONL format
./framesieve stream -input screen -record-dataset dataset.jsonl
```

### 3. VLM Reverse Proxy

Start the reverse proxy to intercept OpenAI or Anthropic vision requests:

```bash
# Start proxy with local mock mode (no API keys required)
./framesieve proxy -listen 0.0.0.0:8088 -mock

# Start proxy forwarding to real OpenAI API
./framesieve proxy -listen 0.0.0.0:8088 -openai https://api.openai.com
```

Configure your autonomous agent to target the local proxy:

```python
import openai

client = openai.OpenAI(
    base_url="http://localhost:8088/v1",
    api_key="your-api-key",
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[
        {
            "role": "user",
            "content": [
                {"type": "text", "text": "Describe the current screen state."},
                {"type": "image_url", "image_url": {"url": "data:image/jpeg;base64,..."}},
            ],
        }
    ],
)
```

Redundant frames sent to the proxy are automatically omitted or cropped, reducing token usage before reaching upstream providers.

---

## Benchmark Results

Run the integrated CPU benchmark suite across all 6 core computer vision routines:

```bash
./framesieve bench
```

Empirical results on standard Intel CPU (Intel Core i5-1135G7 @ 2.40GHz):

| Operation | Resolution | Latency per Frame / Event | Throughput |
|---|---|---|---|
| Perceptual dHash | 1920x1080 (1080p) | 0.112 us | 8,924,667 FPS |
| Dynamic ROI Extraction | 1920x1080 (1080p) | 0.58 ms | 1,726 FPS |
| Multi-ROI Disjoint Clustering | 1920x1080 (1080p) | 0.06 ms | 18,181 FPS |
| SSIM Calculation | 1280x720 (720p) | 1.45 ms | 687 FPS |
| SSIM Calculation | 1920x1080 (1080p) | 3.45 ms | 290 FPS |
| Semantic Delta Classification | 1080p active ROI | 136.3 us | 7,335 events/sec |
| **Total Memory Footprint** | Heap: 5.73 MB | System: 19.90 MB | **PASS** |

---

## Repository Structure

```
framesieve/
├── cmd/
│   └── framesieve/
│       └── main.go                 # CLI entry point (stream, proxy, bench, studio, demo)
├── internal/
│   ├── config/
│   │   └── config.go               # Configuration structures and YAML parser
│   ├── stream/
│   │   ├── engine.go               # Multi-stage perceptual sieve coordinator
│   │   ├── ring.go                 # Thread-safe circular buffer for frame history
│   │   ├── screen_windows.go       # Native Windows GDI screen capture driver
│   │   ├── screen_stub.go          # Cross-platform stub for non-windows builds
│   │   ├── pacer.go                # Token leaky-bucket rate limiter
│   │   ├── synthetic.go            # Synthetic desktop UI interaction generator
│   │   └── ffmpeg.go               # Universal FFmpeg pipe for RTSP and video files
│   ├── vision/
│   │   ├── frame.go                # Frame data model and sync.Pool memory recyclers
│   │   ├── phash.go                # 64-bit dHash gradient calculation
│   │   ├── ssim.go                 # 8x8 block SSIM algorithm
│   │   ├── adaptive.go             # Adaptive noise floor tracker (EMA)
│   │   ├── classifier.go           # Semantic delta classifier (1D Radon profiles)
│   │   ├── multi_roi.go            # Multi-ROI disjoint clustering and composite packer
│   │   ├── gme.go                  # Global motion estimation and jitter compensation
│   │   ├── roi.go                  # Dynamic ROI detection and 28px patch grid alignment
│   │   └── crop.go                 # Zero-copy crop and token cost formulas
│   ├── proxy/
│   │   └── proxy.go                # Reverse proxy for OpenAI and Anthropic vision APIs
│   ├── telemetry/
│   │   ├── store.go                # SQLite metric persistence (modernc.org/sqlite)
│   │   └── dataset.go              # Multimodal JSONL dataset exporter
│   └── server/
│       └── server.go               # HTTP REST and Server-Sent Events (SSE) server
├── web/
│   ├── src/
│   │   ├── App.tsx                 # React 19 root application
│   │   ├── components/             # Monitor canvas (Normal/Heatmap/Grid), charts, timeline
│   │   └── types.ts                # TypeScript type definitions
│   ├── embed.go                    # Go embed packaging for web/dist assets
│   ├── package.json
│   └── vite.config.ts
├── examples/
│   ├── framesieve.yaml             # Production configuration file
│   └── mock_vlm_client.go          # Simulated VLM client sending video sequences
├── docs/
│   ├── architecture.md             # Detailed engineering and algorithm specifications
│   ├── vlm_token_economics.md      # Mathematical token cost comparison
│   └── api_reference.md            # CLI and API documentation
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

## License

MIT License.
