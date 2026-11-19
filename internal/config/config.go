package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// SieveConfig holds the complete configuration parameters for framesieve.
type SieveConfig struct {
	Engine   EngineConfig   `yaml:"engine"`
	Vision   VisionConfig   `yaml:"vision"`
	Proxy    ProxyConfig    `yaml:"proxy"`
	Studio   StudioConfig   `yaml:"studio"`
	Database DatabaseConfig `yaml:"database"`
}

// EngineConfig governs frame ingestion, rate limits, and concurrency.
type EngineConfig struct {
	MaxFPS           int           `yaml:"max_fps"`
	RingBufferSize   int           `yaml:"ring_buffer_size"`
	Workers          int           `yaml:"workers"`
	ProcessingFormat string        `yaml:"processing_format"` // "rgb24", "yuv420p"
	InactivityFlush  time.Duration `yaml:"inactivity_flush"`
}

// VisionConfig configures perceptual hashing, SSIM calculation, and dynamic ROI parameters.
type VisionConfig struct {
	// Fast screening gate
	PHashThreshold int `yaml:"phash_threshold"` // Hamming distance threshold (default 2)

	// Structural Similarity gate
	SSIMThreshold float64 `yaml:"ssim_threshold"` // Values >= threshold are considered redundant (default 0.985)
	LuminanceOnly bool    `yaml:"luminance_only"`

	// Global Motion Estimation
	EnableMotionCompensation bool    `yaml:"enable_motion_compensation"`
	MotionJitterThreshold    float64 `yaml:"motion_jitter_threshold"`

	// Dynamic ROI extraction
	PixelDeltaMin   uint8 `yaml:"pixel_delta_min"`   // Minimal pixel intensity change (default 24)
	MinClusterArea  int   `yaml:"min_cluster_area"`  // Minimal bounding box area in pixels (default 64)
	PaddingMargin   int   `yaml:"padding_margin"`    // Additional margin around bounding box (default 32)
	PatchGridAlign  int   `yaml:"patch_grid_align"`  // Align bounding box to VLM patch grid (e.g. 28)
	MaxRoiExpansion float64 `yaml:"max_roi_expansion"` // If ROI exceeds ratio of full frame, send full frame
}

// ProxyConfig governs the VLM reverse proxy endpoints and upstream targets.
type ProxyConfig struct {
	ListenAddr       string        `yaml:"listen_addr"`
	UpstreamOpenAI   string        `yaml:"upstream_openai"`
	UpstreamAnthropic string       `yaml:"upstream_anthropic"`
	BurstPreFrames   int           `yaml:"burst_pre_frames"`
	BurstPostFrames  int           `yaml:"burst_post_frames"`
	BurstTimeout     time.Duration `yaml:"burst_timeout"`
	OutputImageCodec string        `yaml:"output_image_codec"` // "webp", "jpeg"
	ImageQuality     int           `yaml:"image_quality"`
}

// StudioConfig defines options for the embedded React multimodal console.
type StudioConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Port      int    `yaml:"port"`
	Host      string `yaml:"host"`
	Telemetry bool   `yaml:"telemetry"`
}

// DatabaseConfig specifies the SQLite persistence path.
type DatabaseConfig struct {
	Path string `yaml:"path"`
}

// DefaultConfig returns production-tuned defaults.
func DefaultConfig() *SieveConfig {
	return &SieveConfig{
		Engine: EngineConfig{
			MaxFPS:           30,
			RingBufferSize:   16,
			Workers:          4,
			ProcessingFormat: "rgb24",
			InactivityFlush:  2 * time.Second,
		},
		Vision: VisionConfig{
			PHashThreshold:           2,
			SSIMThreshold:            0.985,
			LuminanceOnly:            true,
			EnableMotionCompensation: true,
			MotionJitterThreshold:    1.5,
			PixelDeltaMin:            24,
			MinClusterArea:           64,
			PaddingMargin:            32,
			PatchGridAlign:           28,
			MaxRoiExpansion:          0.75,
		},
		Proxy: ProxyConfig{
			ListenAddr:        "0.0.0.0:8088",
			UpstreamOpenAI:    "https://api.openai.com",
			UpstreamAnthropic: "https://api.anthropic.com",
			BurstPreFrames:    1,
			BurstPostFrames:   1,
			BurstTimeout:      500 * time.Millisecond,
			OutputImageCodec:  "webp",
			ImageQuality:      85,
		},
		Studio: StudioConfig{
			Enabled:   true,
			Port:      8686,
			Host:      "0.0.0.0",
			Telemetry: true,
		},
		Database: DatabaseConfig{
			Path: "framesieve.db",
		},
	}
}

// LoadConfig loads configuration from a YAML file. If missing, returns default.
func LoadConfig(path string) (*SieveConfig, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
