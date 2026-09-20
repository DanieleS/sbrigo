// Package realtime fans out change events to connected clients over Server-Sent Events.
package realtime

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Event is a change notification. Type follows the "<entity>.<action>" convention, e.g. "task.updated".
type Event struct {
	ID   uint64 `json:"id"`
	Type string `json:"type"`
	Data any    `json:"data"`
}

// Broker is an in-memory publish/subscribe hub. A single container serves the whole family, so
// no external message bus is needed.
type Broker struct {
	mu          sync.RWMutex
	subscribers map[chan Event]struct{}
	seq         atomic.Uint64
	bufferSize  int
	log         *slog.Logger
}

// NewBroker creates a broker; bufferSize is the per-subscriber queue length.
func NewBroker(log *slog.Logger) *Broker {
	return &Broker{subscribers: map[chan Event]struct{}{}, bufferSize: 64, log: log}
}

// Publish delivers the event to every subscriber. Slow subscribers are dropped rather than
// blocking the publisher; they will reconnect and resynchronise.
func (b *Broker) Publish(eventType string, data any) {
	ev := Event{ID: b.seq.Add(1), Type: eventType, Data: data}
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers {
		select {
		case ch <- ev:
		default:
			b.log.Warn("dropping slow sse subscriber")
			go b.unsubscribe(ch)
		}
	}
}

// Subscribe registers a new subscriber; call the returned function to leave.
func (b *Broker) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, b.bufferSize)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	return ch, func() { b.unsubscribe(ch) }
}

func (b *Broker) unsubscribe(ch chan Event) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.subscribers, ch)
}

// SubscriberCount is exposed for tests and diagnostics.
func (b *Broker) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// ServeHTTP streams events to the client until it disconnects.
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache, no-transform")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no") // disable buffering in nginx-style reverse proxies
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, "retry: 3000\nevent: connected\ndata: {}\n\n")
	flusher.Flush()

	events, cancel := b.Subscribe()
	defer cancel()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case ev, ok := <-events:
			if !ok {
				return
			}
			payload, err := json.Marshal(ev.Data)
			if err != nil {
				b.log.Error("marshal sse event", "err", err)
				continue
			}
			if _, err := fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", ev.ID, ev.Type, payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
