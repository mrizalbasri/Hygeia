package care

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/mrizalbasri/Hygeia/backend/internal/notification"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type streamFrame struct{ ID, Event, Data string }

func connectStream(t *testing.T, f *fixture, url, actor, lastID string) (*http.Response, <-chan streamFrame) {
	t.Helper()
	request, _ := http.NewRequest("GET", url, nil)
	request.AddCookie(f.cookies[actor])
	if lastID != "" {
		request.Header.Set("Last-Event-ID", lastID)
	}
	client := &http.Client{Timeout: 12 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { response.Body.Close() })
	frames := make(chan streamFrame, 100)
	if response.StatusCode == 200 {
		go func() {
			defer close(frames)
			scanner := bufio.NewScanner(response.Body)
			var frame streamFrame
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					if frame.Event != "" {
						frames <- frame
					}
					frame = streamFrame{}
					continue
				}
				if strings.HasPrefix(line, "id: ") {
					frame.ID = strings.TrimPrefix(line, "id: ")
				}
				if strings.HasPrefix(line, "event: ") {
					frame.Event = strings.TrimPrefix(line, "event: ")
				}
				if strings.HasPrefix(line, "data: ") {
					frame.Data = strings.TrimPrefix(line, "data: ")
				}
			}
		}()
	}
	return response, frames
}
func nextFrame(t *testing.T, frames <-chan streamFrame) streamFrame {
	t.Helper()
	select {
	case frame, ok := <-frames:
		if !ok {
			t.Fatal("stream closed")
		}
		return frame
	case <-time.After(6 * time.Second):
		t.Fatal("stream frame timeout")
	}
	return streamFrame{}
}
func TestNotificationStreamReplayReauthenticationAndLive(t *testing.T) {
	f := integrationFixture(t)
	e := activeLabEpisode(t, f)
	newPrescription(t, f, e)
	page := notificationPage(t, f, "patient", "/api/v1/notifications")
	if len(page.Notifications) < 2 {
		t.Fatal("missing fixtures")
	}
	server := httptest.NewUnstartedServer(f.api)
	server.Config.WriteTimeout = 100 * time.Millisecond
	server.Start()
	t.Cleanup(server.Close)
	earliest := page.Notifications[len(page.Notifications)-1].ID
	response, frames := connectStream(t, f, server.URL+"/api/v1/notifications/stream?after_id=0", "patient", fmt.Sprint(earliest))
	if response.StatusCode != 200 || response.Header.Get("Content-Type") != "text/event-stream" || response.Header.Get("X-Accel-Buffering") != "no" {
		t.Fatalf("bad SSE headers %v", response.Header)
	}
	frame := nextFrame(t, frames)
	var n notification.Notification
	if err := json.Unmarshal([]byte(frame.Data), &n); err != nil {
		t.Fatal(err)
	}
	if frame.ID != fmt.Sprint(n.ID) || n.ID != page.Notifications[0].ID || frame.Event != "notification" {
		t.Fatal("incorrect cursor replay", frame)
	}
	// This live event is delivered after the server's ordinary WriteTimeout.
	newPrescription(t, f, e)
	live := nextFrame(t, frames)
	if live.Event != "notification" || live.ID == frame.ID {
		t.Fatal("live event missing")
	}
	if strings.Contains(live.Data, "SYNTHETIC-MEDICATION") {
		t.Fatal("clinical data in SSE")
	}
	var unread int
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*) FROM notifications WHERE user_id=$1 AND read_at IS NULL`, f.ids["patient"]).Scan(&unread); err != nil || unread != 3 {
		t.Fatal("SSE changed read state", unread, err)
	}
	if _, err := f.pool.Exec(context.Background(), `DELETE FROM sessions WHERE user_id=$1`, f.ids["patient"]); err != nil {
		t.Fatal(err)
	}
	ended := nextFrame(t, frames)
	if ended.Event != "session-ended" || ended.Data != "{}" {
		t.Fatal("session revocation did not close stream", ended)
	}
}
func TestNotificationStreamIsolationLimitAndCleanup(t *testing.T) {
	f := integrationFixture(t)
	e := activeLabEpisode(t, f)
	newPrescription(t, f, e)
	server := httptest.NewServer(f.api)
	t.Cleanup(server.Close)
	path := server.URL + "/api/v1/notifications/stream"
	one, _ := connectStream(t, f, path, "other", "")
	two, _ := connectStream(t, f, path, "other", "")
	if one.StatusCode != 200 || two.StatusCode != 200 {
		t.Fatal("expected two streams")
	}
	three, _ := connectStream(t, f, path, "other", "")
	if three.StatusCode != 429 {
		t.Fatal("per-user stream limit missing", three.StatusCode)
	}
	// A different user has an independent quota and only receives owned IDs.
	own, frames := connectStream(t, f, path, "patient", "")
	if own.StatusCode != 200 {
		t.Fatal(own.StatusCode)
	}
	page := notificationPage(t, f, "patient", "/api/v1/notifications")
	owned := map[string]bool{}
	for _, n := range page.Notifications {
		owned[fmt.Sprint(n.ID)] = true
	}
	for range page.Notifications {
		if !owned[nextFrame(t, frames).ID] {
			t.Fatal("cross-user notification")
		}
	}
	one.Body.Close()
	deadline := time.Now().Add(2 * time.Second)
	for {
		retry, _ := connectStream(t, f, path, "other", "")
		if retry.StatusCode == 200 {
			retry.Body.Close()
			break
		}
		retry.Body.Close()
		if time.Now().After(deadline) {
			t.Fatal("stream capacity not released")
		}
		time.Sleep(20 * time.Millisecond)
	}
	expectStatus(t, f.request("patient", "GET", "/api/v1/notifications/stream?after_id=-1", "", false), 400)
	expectStatus(t, f.request("patient", "GET", "/api/v1/notifications/stream?user_id="+f.ids["other"], "", false), 400)
	expectStatus(t, f.request("anonymous", "GET", "/api/v1/notifications/stream", "", false), 401)
}
func TestNotificationAllocationCommitOrder(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	first, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Rollback(ctx)
	if err := notification.Enqueue(ctx, first, f.patientID, 700, f.ids["doctor"], "episode.invite", "care_episode", f.patientID); err != nil {
		t.Fatal(err)
	}
	second, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Rollback(ctx)
	done := make(chan error, 1)
	go func() {
		err := notification.Enqueue(ctx, second, f.patientID, 701, f.ids["doctor"], "episode.invite", "care_episode", f.patientID)
		if err == nil {
			err = second.Commit(ctx)
		}
		done <- err
	}()
	select {
	case err := <-done:
		t.Fatalf("later allocation bypassed uncommitted ID: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := first.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	page := notificationPage(t, f, "patient", "/api/v1/notifications")
	if len(page.Notifications) != 2 || page.Notifications[0].ID <= page.Notifications[1].ID {
		t.Fatal("allocation order incorrect")
	}
}
