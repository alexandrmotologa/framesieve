# framesieve

Real-time perceptual video stream compressor and dynamic Region-of-Interest (ROI) extractor for Vision-Language Models.

`framesieve` evaluates video streams (desktop screencasts, webcam feeds, and CCTV streams) on CPU, filtering out redundant frames and extracting active visual deltas before sending payloads to upstream Vision-Language Models (GPT-4o Vision, Gemini 1.5 Pro, Claude Sonnet).

---

## Technical Specifications

- **Processing Latency**: Under 1.5ms per 1080p frame on CPU.
- **Memory Footprint**: Under 15MB RAM steady state via preallocated buffer pools.
- **Token Reduction**: Discards 90% to 95% of static frames, cutting visual token costs by up to 98%.
- **VLM Patch Grid Alignment**: Automatically snaps dynamic crop boundaries to 28px patch multiples, eliminating fractional token waste.
- **Embedded Visual Studio**: Bundles a React 19 studio inside the single Go executable.

---

## Problem Overview

Sending continuous video streams to Vision-Language Models creates severe cost and context window issues:
1. **Token Cost**: A standard 30 FPS stream generates 1,800 frames per minute, consuming over 1.5 million tokens per minute and exhausting context windows in under 60 seconds.
2. **Fixed Subsampling Pitfalls**: Blindly subsampling to 1 FPS misses brief UI transitions, error notifications, and physical anomalies that resolve in 100ms to 350ms.
3. **Background Token Waste**: In desktop agents and fixed camera feeds, 85% to 95% of pixels remain unchanged between frames. Re-transmitting unchanged backgrounds wastes tokens on static content.

`framesieve` solves this by filtering static frames on the CPU and sending only localized bounding box deltas when state transitions occur.

---

## Architecture Pipeline

```
       +-------------------------------------------------------------+
       |             Continuous Video Stream (30 / 60 FPS)           |
       |         (Desktop Screencast, Camera Feed, RTSP / WebRTC)    |
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
       |            SSIM >= 0.985 -> Drop Static Frame               |
       |                                                             |
       |  [Stage 3] Global Motion Estimation (GME)                   |
       |            Camera Jitter Filter (< 1.5 px shift)            |
       |                                                             |
       |  [Stage 4] Dynamic ROI Bounding Box Detector                |
       |            Patch-Grid Alignment (28px tiles)                |
       |            Zero-Copy Sub-Rectangle Extraction               |
       +-------------------------------------------------------------+
                                      |
                       Compact Visual Event Burst
                                      v
       +-------------------------------------------------------------+
       |                Upstream Vision-Language Model               |
       |            (GPT-4o Vision, Gemini 1.5 Pro, Claude)           |
       +-------------------------------------------------------------+
```

1. **Stage 1: 64-bit Perceptual Hash (dHash)**: Samples an 8x8 luminance gradient. A Hamming distance check via the CPU `POPCNT` instruction rejects identical frames in under 0.12 microseconds.
2. **Stage 2: Structural Similarity Index (SSIM)**: Evaluates 8x8 luminance blocks using integer sum accumulators. Frames with SSIM >= 0.985 are dropped in under 0.9 milliseconds.
3. **Stage 3: Global Motion Estimation (GME)**: Computes translational shift vectors across distributed anchor blocks to filter out micro-jitter from handheld cameras or robotics platforms.
4. **Stage 4: Dynamic ROI Crop**: Identifies mutated pixel clusters, adds a 32px context margin, aligns dimensions to 28px patch boundaries, and extracts the crop patch without reallocating full images.

---

## Installation

### Prerequisites

- Go 1.22+ (tested on Go 1.27)
- Node.js 18+ (for building the React 19 studio)
- FFmpeg (optional, required only for video files, RTSP streams, and hardware devices)

### Build from Source

```bash
# Clone the repository
git clone https://github.com/alexandrmotologa/framesieve.git
cd framesieve

# Build embedded web assets and Go binary
make all

# Or build the binary directly if web assets are already built:
go build -o framesieve ./cmd/framesieve
```

---

## Quickstart

### 1. Instant Interactive Demo

Run the built-in synthetic desktop stream and launch the Multimodal Studio on port 8686:

```bash
./framesieve demo
```

Open `http://localhost:8686` in your browser to inspect live SSIM curves, active ROI bounding boxes, and real-time token savings.

### 2. Stream Ingestion

Process an MP4 video recording, live RTSP camera feed, or webcam:

```bash
# Ingest local MP4 recording
./framesieve stream -input recording.mp4 -fps 30

# Ingest live RTSP camera stream
./framesieve stream -input rtsp://192.168.1.100:554/live -fps 30

# Run without the web studio UI
./framesieve stream -input synthetic -no-studio
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

Run the integrated CPU benchmark suite:

```bash
./framesieve bench
```

Empirical results on standard Intel CPU (Intel Core i5-1135G7 @ 2.40GHz):

| Operation | Resolution | Latency per Frame | Throughput |
|---|---|---|---|
| Perceptual dHash | 1920x1080 (1080p) | 0.111 us | 8,976,782 FPS |
| Dynamic ROI Extraction | 1920x1080 (1080p) | 0.71 ms | 1,416 FPS |
| SSIM Calculation | 1280x720 (720p) | 1.42 ms | 703 FPS |
| SSIM Calculation | 1920x1080 (1080p) | 3.19 ms | 314 FPS |
| **Total Memory Footprint** | Heap: 6.06 MB | System: 14.83 MB | **PASS (< 15 MB)** |

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
│   │   ├── synthetic.go            # Synthetic desktop UI interaction generator
│   │   └── ffmpeg.go               # Universal FFmpeg pipe for RTSP and video files
│   ├── vision/
│   │   ├── frame.go                # Frame data model and sync.Pool memory recyclers
│   │   ├── phash.go                # 64-bit dHash gradient calculation
│   │   ├── ssim.go                 # 8x8 block SSIM algorithm
│   │   ├── gme.go                  # Global motion estimation and jitter compensation
│   │   ├── roi.go                  # Dynamic ROI detection and 28px patch grid alignment
│   │   └── crop.go                 # Zero-copy crop and token cost formulas
│   ├── proxy/
│   │   └── proxy.go                # Reverse proxy for OpenAI and Anthropic vision APIs
│   ├── telemetry/
│   │   └── store.go                # SQLite metric persistence (modernc.org/sqlite)
│   └── server/
│       └── server.go               # HTTP REST and Server-Sent Events (SSE) server
├── web/
│   ├── src/
│   │   ├── App.tsx                 # React 19 root application
│   │   ├── components/             # Monitor canvas, SSIM chart, timeline, tuning controls
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
