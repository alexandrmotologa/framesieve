export interface BoundingBox {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface SemanticClassification {
  type: string;
  confidence: number;
  description: string;
}

export interface FrameEvent {
  timestamp: string;
  trigger_frame: {
    id: number;
    is_key?: boolean;
    dropped: boolean;
    drop_code?: string;
    ssim: number;
    latency: number;
  };
  base64_data_uri?: string;
  ssim: number;
  roi?: BoundingBox;
  classification?: SemanticClassification;
  tokens_saved: number;
  tokens_sent: number;
  full_tokens_cost: number;
}

export interface TelemetrySnapshot {
  total_frames: number;
  passed_frames: number;
  dropped_frames: number;
  drop_ratio: number;
  total_tokens_naive: number;
  total_tokens_sent: number;
  tokens_saved: number;
  saved_ratio: number;
  est_dollars_saved: number;
  avg_latency: number; // nanoseconds
  current_fps: number;
  last_event_time: string;
}

export interface SieveConfig {
  engine: {
    max_fps: number;
    ring_buffer_size: number;
  };
  vision: {
    phash_threshold: number;
    ssim_threshold: number;
    padding_margin: number;
    patch_grid_align: number;
  };
}
