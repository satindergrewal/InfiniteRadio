package stream

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewBroadcaster(t *testing.T) {
	b := NewBroadcaster()
	if b == nil {
		t.Fatal("NewBroadcaster returned nil")
	}
	if b.ListenerCount() != 0 {
		t.Errorf("Initial ListenerCount = %d, want 0", b.ListenerCount())
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	b := NewBroadcaster()

	l1 := b.Subscribe()
	if b.ListenerCount() != 1 {
		t.Errorf("After 1 subscribe: ListenerCount = %d, want 1", b.ListenerCount())
	}

	l2 := b.Subscribe()
	if b.ListenerCount() != 2 {
		t.Errorf("After 2 subscribes: ListenerCount = %d, want 2", b.ListenerCount())
	}

	b.Unsubscribe(l1)
	if b.ListenerCount() != 1 {
		t.Errorf("After 1 unsubscribe: ListenerCount = %d, want 1", b.ListenerCount())
	}

	b.Unsubscribe(l2)
	if b.ListenerCount() != 0 {
		t.Errorf("After all unsubscribed: ListenerCount = %d, want 0", b.ListenerCount())
	}
}

func TestBroadcastDelivers(t *testing.T) {
	b := NewBroadcaster()
	l := b.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	source := make(chan []int16, 10)

	go b.Run(ctx, source)

	// Send a frame
	frame := []int16{100, 200, 300, 400}
	source <- frame

	// Listener should receive it
	select {
	case got := <-l.C:
		if len(got) != len(frame) {
			t.Errorf("Received frame length %d, want %d", len(got), len(frame))
		}
		for i, v := range got {
			if v != frame[i] {
				t.Errorf("Frame[%d] = %d, want %d", i, v, frame[i])
			}
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for frame")
	}

	cancel()
	b.Unsubscribe(l)
}

func TestBroadcastMultipleListeners(t *testing.T) {
	b := NewBroadcaster()
	listeners := make([]*Listener, 5)
	for i := range listeners {
		listeners[i] = b.Subscribe()
	}

	ctx, cancel := context.WithCancel(context.Background())
	source := make(chan []int16, 10)

	go b.Run(ctx, source)

	frame := []int16{42, -42}
	source <- frame

	// All listeners should get the frame
	for i, l := range listeners {
		select {
		case got := <-l.C:
			if got[0] != 42 {
				t.Errorf("Listener %d got frame[0]=%d, want 42", i, got[0])
			}
		case <-time.After(time.Second):
			t.Errorf("Listener %d timed out", i)
		}
	}

	cancel()
	for _, l := range listeners {
		b.Unsubscribe(l)
	}
}

func TestBroadcastDropsSlowListener(t *testing.T) {
	b := NewBroadcaster()
	slow := b.Subscribe()
	fast := b.Subscribe()

	ctx, cancel := context.WithCancel(context.Background())
	source := make(chan []int16, 200)

	go b.Run(ctx, source)

	// Fill the slow listener's buffer (150 capacity) without reading
	for i := 0; i < 200; i++ {
		source <- []int16{int16(i)}
	}

	// Give broadcaster time to process
	time.Sleep(100 * time.Millisecond)

	// Fast listener should have frames (we drain them)
	fastCount := 0
	for {
		select {
		case <-fast.C:
			fastCount++
		default:
			goto done
		}
	}
done:

	// Slow listener should have exactly buffer capacity (150) frames, rest dropped
	slowCount := 0
	for {
		select {
		case <-slow.C:
			slowCount++
		default:
			goto countDone
		}
	}
countDone:

	if slowCount > 150 {
		t.Errorf("Slow listener got %d frames, should cap at buffer size 150", slowCount)
	}
	if fastCount == 0 {
		t.Error("Fast listener got 0 frames")
	}

	cancel()
	b.Unsubscribe(slow)
	b.Unsubscribe(fast)
}

func TestBroadcastStopsOnContextCancel(t *testing.T) {
	b := NewBroadcaster()
	ctx, cancel := context.WithCancel(context.Background())
	source := make(chan []int16, 10)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.Run(ctx, source)
	}()

	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// good
	case <-time.After(2 * time.Second):
		t.Fatal("Broadcaster did not stop after context cancel")
	}
}

func TestBroadcastStopsOnSourceClose(t *testing.T) {
	b := NewBroadcaster()
	ctx := context.Background()
	source := make(chan []int16, 10)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		b.Run(ctx, source)
	}()

	close(source)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// good
	case <-time.After(2 * time.Second):
		t.Fatal("Broadcaster did not stop after source closed")
	}
}

func TestListenerDoneChannel(t *testing.T) {
	b := NewBroadcaster()
	l := b.Subscribe()

	b.Unsubscribe(l)

	// done channel should be closed
	select {
	case <-l.done:
		// good
	default:
		t.Error("Listener done channel not closed after unsubscribe")
	}
}
