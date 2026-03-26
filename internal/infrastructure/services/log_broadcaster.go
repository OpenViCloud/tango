package services

import (
	"strings"
	"sync"
)

// LogBroadcaster is an io.Writer that fans out writes to all live subscribers
// and keeps a replay buffer so late-joiners receive past output.
type LogBroadcaster struct {
	mu     sync.Mutex
	buf    strings.Builder
	subs   map[chan []byte]struct{}
	closed bool
}

func newLogBroadcaster() *LogBroadcaster {
	return &LogBroadcaster{subs: make(map[chan []byte]struct{})}
}

// Write implements io.Writer — called by exec.Cmd output and appendLog.
func (b *LogBroadcaster) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf.Write(p)
	cp := make([]byte, len(p))
	copy(cp, p)
	for ch := range b.subs {
		select {
		case ch <- cp:
		default: // slow consumer — drop chunk rather than block
		}
	}
	return len(p), nil
}

// Subscribe returns a read channel that receives log chunks and an unsubscribe
// func the caller must invoke when done. The channel is pre-seeded with the
// current replay buffer before any new chunks arrive.
func (b *LogBroadcaster) Subscribe() (<-chan []byte, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan []byte, 512)
	if b.closed {
		close(ch)
		return ch, func() {}
	}
	if b.buf.Len() > 0 {
		replay := []byte(b.buf.String())
		ch <- replay
	}
	b.subs[ch] = struct{}{}
	unsub := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		if _, ok := b.subs[ch]; ok {
			delete(b.subs, ch)
			close(ch)
		}
	}
	return ch, unsub
}

// Snapshot returns the full buffered log as a string (for DB persistence).
func (b *LogBroadcaster) Snapshot() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// closeAll signals end-of-stream to all subscribers by closing their channels.
func (b *LogBroadcaster) closeAll() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	for ch := range b.subs {
		close(ch)
		delete(b.subs, ch)
	}
}
