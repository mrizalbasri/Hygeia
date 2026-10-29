package care

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNotificationStreamGracefulShutdown(t *testing.T) {
	f := integrationFixture(t)
	started := make(chan struct{})
	release := make(chan struct{}, 1)
	defer close(release)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/slow" {
			f.api.ServeHTTP(w, r)
			return
		}
		close(started)
		select {
		case <-release:
			w.WriteHeader(200)
		case <-r.Context().Done():
			w.WriteHeader(500)
		}
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := &http.Client{Timeout: 5 * time.Second}
	ordinary := make(chan int, 1)
	go func() {
		response, err := client.Get(server.URL + "/slow")
		if err != nil {
			ordinary <- 0
			return
		}
		response.Body.Close()
		ordinary <- response.StatusCode
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("ordinary request did not start")
	}
	response, frames := connectStream(t, f, server.URL+"/api/v1/notifications/stream", "other", "")
	if response.StatusCode != 200 {
		t.Fatal("stream did not connect", response.StatusCode)
	}
	// Production invokes this cancellation via its signal context, before Shutdown.
	f.stopStreams()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stopped := make(chan error, 1)
	go func() { stopped <- server.Config.Shutdown(ctx) }()
	select {
	case _, open := <-frames:
		if open {
			t.Fatal("shutdown stream emitted unexpected data")
		}
	case <-ctx.Done():
		t.Fatal("SSE did not close on shutdown")
	}
	select {
	case status := <-ordinary:
		t.Fatalf("stream shutdown interrupted ordinary request: %d", status)
	default:
	}
	release <- struct{}{}
	select {
	case status := <-ordinary:
		if status != 200 {
			t.Fatal("ordinary request failed to drain", status)
		}
	case <-ctx.Done():
		t.Fatal("ordinary request stuck")
	}
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatal("graceful shutdown failed", err)
		}
	case <-ctx.Done():
		t.Fatal("server shutdown timed out")
	}
}
