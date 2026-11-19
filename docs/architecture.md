# framesieve System Architecture

## Overview

`framesieve` is an edge perceptual frame compressor designed for Vision-Language Model (VLM) ingestion pipelines. It processes continuous video streams (desktop screencasts, webcam feeds, and RTSP streams) on the CPU, removing redundant frames and extracting localized dynamic regions of interest before frames are transmitted to upstream models.

The system is engineered in Go with SIMD-aligned image loops, zero-allocation memory pools, and an embedded React 19 visual studio.

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

## 1. Multi-Stage Perceptual Sieve Pipeline

### Stage 1: 64-bit Difference Hashing (dHash)
The first screening gate runs in under 0.12 microseconds per frame. The 8-bit ITU-R BT.601 luminance channel is downscaled across a 9-column by 8-row grid:

```
Y = (19595 * R + 38470 * G + 7471 * B) >> 16
```

Adjacent horizontal pixels are compared across 8 rows:
- If `Luminance[x, y] > Luminance[x + 1, y]`, the corresponding bit is set to 1.
- The resulting 64-bit unsigned integer is compared against the reference frame hash via `bits.OnesCount64(hashA ^ hashB)`.
- If the Hamming distance is less than or equal to the threshold (default: 2), the frame is dropped immediately without evaluating SSIM.

### Stage 2: Structural Similarity Index (SSIM)
Frames passing the dHash gate are evaluated using an 8x8 block-based SSIM algorithm:

$$\text{SSIM}(\mathbf{x}, \mathbf{y}) = \frac{(2\mu_x \mu_y + C_1)(2\sigma_{xy} + C_2)}{(\mu_x^2 + \mu_y^2 + C_1)(\sigma_x^2 + \sigma_y^2 + C_2)}$$

Constants are calibrated for 8-bit dynamic range ($L = 255$):
- $C_1 = (0.01 \times 255)^2 = 6.5025$
- $C_2 = (0.03 \times 255)^2 = 58.5225$

Integer accumulators compute $\sum x, \sum y, \sum x^2, \sum y^2,$ and $\sum xy$ in single passes over contiguous 64-pixel blocks. If the mean SSIM across blocks is greater than or equal to the threshold (default: 0.985), the frame is dropped as redundant.

### Stage 3: Global Motion Estimation (GME)
In screencasts or robotics streams, camera jitter or page scrolling causes widespread pixel differences without introducing new semantic objects.

The engine uses five distributed anchor blocks (center and four quadrants) to evaluate translational displacement $(dx, dy)$ over a search window. If the motion confidence is above 0.85 and the compensated residual difference is below the jitter threshold (default: 1.5), the frame is marked as motion jitter and discarded.

### Stage 4: Dynamic Region of Interest (ROI) Extraction
When a verified state change occurs:
1. Pixel intensity difference $D(x, y) = |y(x, y) - x(x, y)|$ is calculated.
2. Active clusters exceeding intensity delta 24 are enclosed within minimal bounding rectangle $[x_{\min}, y_{\min}, x_{\max}, y_{\max}]$.
3. A contextual padding margin (32px) is added.
4. Dimensions are aligned to multiples of 28px, ensuring exact correspondence to VLM token tile boundaries.
5. If the bounding box area exceeds 75% of the total frame, the full frame is forwarded to preserve peripheral context.

---

## 2. Memory Architecture and Zero-Allocation Pools

High frame-rate ingestion can trigger continuous garbage collector pauses if memory buffers are allocated dynamically.

`framesieve` uses `sync.Pool` structures within `vision.FramePool`:
- Preallocated RGB24 byte slices (`Width * Height * 3`).
- Preallocated 8-bit luminance byte slices (`Width * Height`).
- Thread-safe ring buffer (`stream.RingBuffer`) holding 16 historical frames.
- Expired frames are returned to the pool, maintaining steady-state memory utilization below 15 MB RAM.

---

## 3. Reverse Proxy Architecture

The reverse proxy exposes standard endpoints:
- `POST /v1/chat/completions` (OpenAI format)
- `POST /v1/messages` (Anthropic format)

Incoming image sequences (Base64 Data URIs) are decoded into the vision pipeline. Redundant frames are omitted or replaced with reference placeholders, and localized deltas are replaced with cropped patches along with spatial coordinate tags (`[Dynamic ROI: X=..., Y=..., W=..., H=...]`).
