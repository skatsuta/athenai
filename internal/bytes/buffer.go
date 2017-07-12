// Package bytes provides goroutine-safe Buffer for testing-only purpose.
package bytes

import (
	"bytes"
	"sync"
)

// A Buffer is a variable-sized buffer of bytes with Read and Write methods.
// The zero value for Buffer is an empty buffer ready to use.
// This is goroutine-safe for Write and String method.
type Buffer struct {
	mu sync.Mutex
	bytes.Buffer
}

func (b *Buffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.Write(p)
}

func (b *Buffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.String()
}

// Close does nothing. Just implements io.Closer.
func (b *Buffer) Close() error {
	return nil
}
