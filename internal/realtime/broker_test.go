package realtime

import (
	"bufio"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBrokerFanOutAndUnsubscribe(t *testing.T) {
	b := NewBroker(slog.Default())
	c1, cancel1 := b.Subscribe()
	c2, cancel2 := b.Subscribe()
	defer cancel2()

	b.Publish("task.created", map[string]string{"id": "1"})
	for _, ch := range []<-chan Event{c1, c2} {
		select {
		case ev := <-ch:
			if ev.Type != "task.created" || ev.ID != 1 {
				t.Fatalf("unexpected event %+v", ev)
			}
		case <-time.After(time.Second):
			t.Fatal("event not delivered")
		}
	}
	cancel1()
	if got := b.SubscriberCount(); got != 1 {
		t.Fatalf("subscribers=%d want 1", got)
	}
}

func TestBrokerDropsSlowSubscriber(t *testing.T) {
	b := NewBroker(slog.Default())
	b.bufferSize = 1
	_, cancel := b.Subscribe()
	defer cancel()
	b.Publish("a", nil)
	b.Publish("b", nil) // buffer full: subscriber is evicted asynchronously
	deadline := time.Now().Add(time.Second)
	for b.SubscriberCount() != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if b.SubscriberCount() != 0 {
		t.Fatal("slow subscriber was not dropped")
	}
}

func TestServeHTTPStreamsEvents(t *testing.T) {
	b := NewBroker(slog.Default())
	srv := httptest.NewServer(b)
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type %q", ct)
	}

	// Wait for the subscription to be registered before publishing.
	deadline := time.Now().Add(time.Second)
	for b.SubscriberCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	b.Publish("task.updated", map[string]any{"id": "abc", "is_completed": true})

	var lines []string
	r := bufio.NewReader(resp.Body)
	for len(lines) < 8 {
		line, err := r.ReadString('\n')
		if err != nil {
			t.Fatalf("read: %v (lines so far: %q)", err, lines)
		}
		lines = append(lines, strings.TrimRight(line, "\n"))
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"event: connected", "event: task.updated", `data: {"id":"abc","is_completed":true}`, "id: 1"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("stream missing %q:\n%s", want, joined)
		}
	}
}
