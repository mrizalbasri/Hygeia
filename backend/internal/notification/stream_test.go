package notification

import (
	"context"
	"testing"
	"time"
)

func TestHeartbeatWaitsForFreshPoll(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	poll, heartbeat := make(chan time.Time), make(chan time.Time)
	writes := make(chan string, 1)
	done := make(chan bool, 1)
	go func() {
		done <- waitForPoll(ctx, poll, heartbeat, func(frame string) bool { writes <- frame; return true })
	}()
	heartbeat <- time.Now()
	if frame := <-writes; frame != ": heartbeat\n\n" {
		t.Fatal(frame)
	}
	select {
	case <-done:
		t.Fatal("heartbeat advanced the consumed notification batch")
	default:
	}
	poll <- time.Now()
	if !<-done {
		t.Fatal("poll failed to fetch fresh batch")
	}
}
func TestHeartbeatWriteFailureCloses(t *testing.T) {
	heartbeat := make(chan time.Time, 1)
	heartbeat <- time.Now()
	if waitForPoll(context.Background(), nil, heartbeat, func(string) bool { return false }) {
		t.Fatal("failed write kept stream alive")
	}
}
