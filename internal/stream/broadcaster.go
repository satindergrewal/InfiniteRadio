package stream

import (
	"context"
	"sync"
)

// Broadcaster fans out PCM frames from one source to N listeners.
type Broadcaster struct {
	mu        sync.RWMutex
	listeners map[*Listener]struct{}
}

// Listener receives PCM frames from the broadcaster.
type Listener struct {
	C    chan []int16   // buffered channel of 20ms PCM frames
	done chan struct{}
}

// NewBroadcaster creates a new broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		listeners: make(map[*Listener]struct{}),
	}
}

// Subscribe registers a new listener. Returns a Listener that receives frames.
func (b *Broadcaster) Subscribe() *Listener {
	l := &Listener{
		C:    make(chan []int16, 150), // ~3 seconds of buffer at 20ms/frame
		done: make(chan struct{}),
	}
	b.mu.Lock()
	b.listeners[l] = struct{}{}
	b.mu.Unlock()
	return l
}

// Unsubscribe removes a listener and signals it to stop.
func (b *Broadcaster) Unsubscribe(l *Listener) {
	b.mu.Lock()
	delete(b.listeners, l)
	b.mu.Unlock()
	close(l.done)
}

// ListenerCount returns the number of active listeners.
func (b *Broadcaster) ListenerCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.listeners)
}

// Run reads frames from source and fans out to all listeners.
// Slow listeners get frames dropped rather than blocking the broadcast.
func (b *Broadcaster) Run(ctx context.Context, source <-chan []int16) {
	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-source:
			if !ok {
				return
			}
			b.mu.RLock()
			for l := range b.listeners {
				select {
				case l.C <- frame:
				default:
					// listener too slow, drop frame to keep broadcast moving
				}
			}
			b.mu.RUnlock()
		}
	}
}
