package vision

import (
	"image"
	"sync"
	"time"
)

// BoundingBox defines an active region of interest in pixel coordinates.
type BoundingBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Area returns the total pixel area of the bounding box.
func (b BoundingBox) Area() int {
	return b.Width * b.Height
}

// Point represents a 2D vector for motion offset.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Frame represents a video frame with cached luminance and perceptual descriptors.
type Frame struct {
	ID        int64         `json:"id"`
	Timestamp time.Time     `json:"timestamp"`
	Width     int           `json:"width"`
	Height    int           `json:"height"`
	Data      []byte        `json:"-"` // Raw RGB24 payload: Width * Height * 3
	Luminance []byte        `json:"-"` // Grayscale 8-bit luminance: Width * Height
	PHash     uint64        `json:"phash"`
	SSIM      float64       `json:"ssim"`
	Motion    Point         `json:"motion"`
	ROI       *BoundingBox  `json:"roi,omitempty"`
	Cropped   []byte        `json:"-"` // Encoded cropped patch (WebP/JPEG)
	IsKey     bool          `json:"is_key"`
	Dropped   bool          `json:"dropped"`
	DropCode  string        `json:"drop_code,omitempty"` // "phash_static", "ssim_static", "motion_jitter"
	Latency   time.Duration `json:"latency"`
}

// FramePool manages recyclable frame objects and byte buffers.
type FramePool struct {
	framePool sync.Pool
	dataPool  sync.Pool
	lumPool   sync.Pool
	width     int
	height    int
}

// NewFramePool creates a pool configured for specific video dimensions.
func NewFramePool(width, height int) *FramePool {
	return &FramePool{
		width:  width,
		height: height,
		framePool: sync.Pool{
			New: func() any {
				return &Frame{}
			},
		},
		dataPool: sync.Pool{
			New: func() any {
				return make([]byte, width*height*3)
			},
		},
		lumPool: sync.Pool{
			New: func() any {
				return make([]byte, width*height)
			},
		},
	}
}

// Get acquires a clean Frame with preallocated buffers.
func (p *FramePool) Get() *Frame {
	f := p.framePool.Get().(*Frame)
	f.Width = p.width
	f.Height = p.height
	if cap(f.Data) < p.width*p.height*3 {
		f.Data = p.dataPool.Get().([]byte)
	} else {
		f.Data = f.Data[:p.width*p.height*3]
	}
	if cap(f.Luminance) < p.width*p.height {
		f.Luminance = p.lumPool.Get().([]byte)
	} else {
		f.Luminance = f.Luminance[:p.width*p.height]
	}
	f.ID = 0
	f.Timestamp = time.Time{}
	f.PHash = 0
	f.SSIM = 1.0
	f.Motion = Point{}
	f.ROI = nil
	f.Cropped = nil
	f.IsKey = false
	f.Dropped = false
	f.DropCode = ""
	f.Latency = 0
	return f
}

// Put returns the Frame and its slices back to the pool.
func (p *FramePool) Put(f *Frame) {
	if f == nil {
		return
	}
	if f.Data != nil && len(f.Data) == p.width*p.height*3 {
		p.dataPool.Put(f.Data)
	}
	if f.Luminance != nil && len(f.Luminance) == p.width*p.height {
		p.lumPool.Put(f.Luminance)
	}
	f.Data = nil
	f.Luminance = nil
	f.Cropped = nil
	f.ROI = nil
	p.framePool.Put(f)
}

// PopulateLuminance converts RGB24 to 8-bit ITU-R BT.601 luminance: Y = 0.299R + 0.587G + 0.114B
// Integer arithmetic optimization: (19595*R + 38470*G + 7471*B) >> 16
func (f *Frame) PopulateLuminance() {
	if len(f.Luminance) < f.Width*f.Height || len(f.Data) < f.Width*f.Height*3 {
		return
	}
	src := f.Data
	dst := f.Luminance
	total := f.Width * f.Height

	for i := 0; i < total; i++ {
		si := i * 3
		r := uint32(src[si])
		g := uint32(src[si+1])
		b := uint32(src[si+2])
		dst[i] = uint8((19595*r + 38470*g + 7471*b) >> 16)
	}
}

// ToRGBA converts the frame RGB24 buffer into standard Go image.RGBA for encoding/rendering.
func (f *Frame) ToRGBA() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
	src := f.Data
	dst := img.Pix
	total := f.Width * f.Height

	for i := 0; i < total; i++ {
		si := i * 3
		di := i * 4
		dst[di] = src[si]
		dst[di+1] = src[si+1]
		dst[di+2] = src[si+2]
		dst[di+3] = 255
	}
	return img
}
