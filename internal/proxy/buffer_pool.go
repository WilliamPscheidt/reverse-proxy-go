package proxy

import "sync"

const (
	// DefaultBufferSize is 32KB, standard optimized buffer size for TCP proxying.
	DefaultBufferSize = 32 * 1024
)

// BufferPool manages reusable byte slices to minimize GC pressure during streaming.
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new BufferPool with the given buffer size.
func NewBufferPool(size int) *BufferPool {
	if size <= 0 {
		size = DefaultBufferSize
	}
	return &BufferPool{
		pool: sync.Pool{
			New: func() any {
				b := make([]byte, size)
				return &b
			},
		},
	}
}

// Get retrieves a buffer pointer from the pool.
func (p *BufferPool) Get() *[]byte {
	return p.pool.Get().(*[]byte)
}

// Put returns a buffer pointer back to the pool.
func (p *BufferPool) Put(b *[]byte) {
	p.pool.Put(b)
}
