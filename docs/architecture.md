# framesieve System Architecture

## Overview

`framesieve` is an edge perceptual frame compressor designed for Vision-Language Model (VLM) ingestion pipelines. It processes continuous video streams (desktop screencasts, webcam feeds, and RTSP streams) on the CPU, removing redundant frames and extracting localized dynamic regions of interest before frames are transmitted to upstream models.

The system is engineered in Go with SIMD-aligned image loops, zero-allocation memory pools, and an embedded React 19 visual studio.

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

## 1. Advanced Vision Pipeline Modules

### 1.1 64-bit Difference Hashing (dHash)
The initial gate runs in 0.11 microseconds per 1080p frame. The luminance plane is sampled across 9 horizontal and 8 vertical coordinates:
- If `Luminance[x, y] > Luminance[x + 1, y]`, bit is 1.
- Evaluated via CPU `POPCNT` instruction (`math/bits.OnesCount64`).

### 1.2 8x8 Block SSIM with Adaptive Noise Floor Calibration
Calculates structural similarity over 8x8 blocks on ITU-R BT.601 luminance:

$$\text{SSIM}(\mathbf{x}, \mathbf{y}) = \frac{(2\mu_x \mu_y + C_1)(2\sigma_{xy} + C_2)}{(\mu_x^2 + \mu_y^2 + C_1)(\sigma_x^2 + \sigma_y^2 + C_2)}$$

The `AdaptiveNoiseTracker` models ambient sensor grain and video compression ringing using an Exponential Moving Average (EMA) of mean absolute differences across static areas:
$$\text{EffSSIM} = \min(0.995, \max(0.940, \text{BaseSSIM} - (\text{MAD}_{\text{EMA}} - 2.0) \times 0.002))$$
This prevents low-light sensor snow or MPEG macroblocking from triggering false event spikes.

### 1.3 Multi-ROI Disjoint Cluster Extraction & Packing
When multiple disconnected regions change simultaneously (e.g., top status bar and bottom chat window), a single enclosing rectangle would capture the entire screen.

`DetectMultiROI` segments connected active cells on a 32x32 grid using 8-connectivity flood-fill. If multiple clusters are separated, the engine extracts disjoint bounding boxes and packs them into a single vertical composite image strip via `PackCompositeROIs`, saving up to 50% more tokens than an all-encompassing bounding box.

### 1.4 Semantic Delta Classification
When a region of interest mutates, `ClassifySemanticDelta` calculates 1D Radon projection profiles along the horizontal axis:
$$H(y) = \sum_{x} \mathbb{I}(|y(x, y) - x(x, y)| \ge \tau_{\text{pixel}})$$
- **TEXT_INPUT**: Alternating peaks and valleys matching font line heights and spacing gaps.
- **MODAL_POPUP**: High fill density (> 48%) in a rectangular geometry.
- **BUTTON_CLICK**: Compact dimensions (< 180x70px) with rapid localized mutation.
- **SCROLL_TRANSLATION**: Vertical motion displacement ($|dy| > 4$) across tall vertical strips.
- **MEDIA_PLAYBACK**: High-frequency unstructured mutation covering large areas.

### 1.5 Token Leaky Bucket Pacer
The `TokenPacer` prevents rapid UI animations or high-action videos from exhausting token quotas:
- Enforces user-configured `MaxTokensPerMinute` and minimum inter-burst cooldown interval (default: 200ms).
- Provides an emergency bypass for extreme visual shocks (SSIM < 0.70).

---

## 2. Ingestion & Persistence

### 2.1 Native Desktop Capture (Zero External Dependencies)
On Windows, `IngestNativeScreen` captures the primary desktop directly through Win32 GDI calls (`GetDC`, `CreateCompatibleBitmap`, `BitBlt`, `GetDIBits`), eliminating the need to install FFmpeg for desktop screen recording.

### 2.2 Multimodal Dataset Recorder
The `DatasetRecorder` streams structured event observations into a `.jsonl` file:
- Base64 cropped images.
- Semantic classification tags.
- Spatial bounding box coordinates.
- Preformatted prompt annotations ready for multimodal SFT fine-tuning and Vision RAG.
