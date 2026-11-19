package stream

import (
	"sync"

	"github.com/alexandrmotologa/framesieve/internal/vision"
)

// RingBuffer implements a thread-safe circular buffer holding a rolling temporal window of frames.
type RingBuffer struct {
	mu       sync.RWMutex
	frames   []*vision.Frame
	capacity int
	head     int
	count    int
	pool     *vision.FramePool
}

// NewRingBuffer allocates a ring buffer with fixed capacity.
func NewRingBuffer(capacity int, pool *vision.FramePool) *RingBuffer {
	if capacity <= 0 {
		capacity = 16
	}
	return &RingBuffer{
		frames:   make([]*vision.Frame, capacity),
		capacity: capacity,
		pool:     pool,
	}
}

// Push appends a new frame into the ring buffer.
// If the buffer is full, the oldest frame is recycled back into the pool.
func (r *RingBuffer) Push(f *vision.Frame) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.count == r.capacity {
		oldest := r.frames[r.head]
		if oldest != nil && r.pool != nil {
			r.pool.Put(oldest)
		}
	} else {
		r.count++
	}

	r.frames[r.head] = f
	r.head = (r.head + 1) % r.capacity
}

// Recent returns the last n frames in chronological order.
func (r *RingBuffer) Recent(n int) []*vision.Frame {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if n > r.count {
		n = r.count
	}
	if n <= 0 {
		return nil
	}

	res := make([]*vision.Frame, n)
	startIndex := (r.head - n + r.capacity) % r.capacity

	for i := 0; i < n; i++ {
		idx := (startIndex + i) % r.capacity
		res[i] = r.frames[idx]
	}

	return res
}

// Latest returns the most recently pushed frame or nil if empty.
func (r *RingBuffer) Latest() *vision.Frame {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.count == 0 {
		return nil
	}

	idx := (r.head - 1 + r.capacity) % r.capacity
	return r.frames[idx]
}

// Clear flushes all frames in the buffer and recycles them.
func (r *RingBuffer) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := 0; i < r.capacity; i++ {
		if r.frames[i] != nil {
			if r.pool != nil {
				r.pool.Put(r.frames[i])
			}
			r.frames[i] = nil
		}
	}
	r.head = 0
	r.count = 0
}
