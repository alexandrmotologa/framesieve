# framesieve: Architecture and Engineering Master Plan
> Real-Time VLM Stream Compressor, Perceptual Frame Sieve & Dynamic Region-of-Interest Optimizer
> Target: Ultra-Fast Go Edge Engine (< 15MB RAM, < 2ms per frame) with Embedded React 19 Multimodal Studio

---

## 1. Problem Statement and Motivation

### 1.1 The Economics of Video Ingestion in Vision-Language Models
Vision-Language Models (VLMs) such as GPT-4o Vision, Gemini 1.5 Pro, and Claude Sonnet process continuous video by tiling frames into discrete patches (tokens):
1. **Token Cost Explosion**: A 30 FPS video generates 1,800 frames per minute, consuming over 1.5 million tokens per minute and exhausting model context windows in under 60 seconds at costs exceeding $290 per operational hour.
2. **Failure of Naive Fixed Subsampling**: Subsampling at fixed intervals (e.g., 1 FPS) misses rapid UI state transitions, dialogs, and error messages that occur in 100ms to 350ms windows.
3. **Redundant Background Token Waste**: In desktop screencasts and static cameras, 85% to 95% of pixels remain unchanged between consecutive frames. Re-transmitting unchanged backgrounds wastes tokens on static content.

### 1.2 The framesieve Solution
`framesieve` provides a local perceptual frame sieve written in Go with SIMD-aligned image loops:
- **Zero-Allocation Structural Similarity (SSIM) Filtering**: Discards 90% to 95% of static frames on CPU in under 1.5ms per frame.
- **Dynamic Region-of-Interest (ROI) Cropping**: Isolates active delta clusters, adds contextual margins, and aligns dimensions to 28px VLM patch grid boundaries.
- **Global Motion Estimation (GME)**: Distinguishes uniform camera jitter from semantic mutations.
- **Event-Driven Keyframe Burst Dispatcher**: Dispatches pre-event context, delta crops, and settling frames.
- **VLM Reverse Proxy**: Transparently intercepts OpenAI and Anthropic vision chat endpoints.
- **Embedded React 19 Multimodal Studio**: Bundles an interactive monitoring dashboard directly inside the Go binary via `go:embed`.

---

## 2. Multi-Stage Filtering Pipeline

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

---

## 3. Mathematical Formulation

### 3.1 Structural Similarity Index (SSIM)
Evaluated across 8x8 blocks on ITU-R BT.601 luminance:

$$\text{SSIM}(\mathbf{x}, \mathbf{y}) = \frac{(2\mu_x \mu_y + C_1)(2\sigma_{xy} + C_2)}{(\mu_x^2 + \mu_y^2 + C_1)(\sigma_x^2 + \sigma_y^2 + C_2)}$$

Constants: $C_1 = 6.5025$, $C_2 = 58.5225$.

### 3.2 Dynamic ROI Bounding Box & Patch Alignment
For pixel delta map $\mathbf{D} = |\mathbf{y} - \mathbf{x}| \ge \tau_{\text{pixel}}$:
- Bounding rectangle $[x_{\min}, y_{\min}, x_{\max}, y_{\max}]$ expanded by contextual margin $\delta = 32\text{px}$.
- Clamped and aligned to $w_{\text{aligned}} = \lceil w / 28 \rceil \times 28$ and $h_{\text{aligned}} = \lceil h / 28 \rceil \times 28$.
- Max expansion ratio threshold: 0.75.

---

## 4. Implementation Status

- [x] **Stage 1: Core Vision & Perceptual Filters** (`internal/vision`)
  - [x] Zero-allocation frame pool (`FramePool`) with preallocated buffers.
  - [x] 64-bit dHash gradient calculation and POPCNT Hamming distance.
  - [x] 8x8 block-based SSIM calculation without heap allocations.
  - [x] Global Motion Estimation for jitter rejection.
  - [x] Dynamic ROI detector with 28px VLM patch grid alignment.
  - [x] Zero-copy crop and visual token estimation.
- [x] **Stage 2: Stream Ingestion & Processing Pipeline** (`internal/stream`)
  - [x] Thread-safe ring buffer (`RingBuffer`) for temporal history.
  - [x] Synthetic 30 FPS desktop stream generator for automated testing.
  - [x] Universal FFmpeg pipe ingestion for local files and RTSP.
  - [x] Event burst coordinator and telemetry tracking.
- [x] **Stage 3: Telemetry & Persistence** (`internal/telemetry`)
  - [x] Zero-CGO SQLite persistence via `modernc.org/sqlite`.
  - [x] Frame-by-frame and session-level metrics storage.
- [x] **Stage 4: VLM Reverse Proxy** (`internal/proxy`)
  - [x] Transparent proxy for `/v1/chat/completions` and `/v1/messages`.
  - [x] Perceptual frame filtering and dynamic ROI substitution.
  - [x] Offline mock mode for testing without API keys.
- [x] **Stage 5: Multimodal Studio Web Console** (`web/` & `internal/server`)
  - [x] React 19 + TypeScript + Vite + Tailwind CSS frontend.
  - [x] Live Canvas monitor with ROI overlay.
  - [x] SSIM curve chart and token savings hero cards.
  - [x] Keyframe burst timeline filmstrip.
  - [x] Static asset embedding via Go `go:embed`.
- [x] **Stage 6: CLI & Benchmarking** (`cmd/framesieve`)
  - [x] Subcommands: `stream`, `proxy`, `bench`, `studio`, `demo`.
  - [x] Verified CPU latency < 1.5ms and RAM < 15MB.
