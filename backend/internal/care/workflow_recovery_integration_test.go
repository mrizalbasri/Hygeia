package care

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mrizalbasri/Hygeia/backend/internal/laboratory"
	"github.com/mrizalbasri/Hygeia/backend/internal/timeline"
)

func TestCorrectionNotifiesAndAppearsInTimelineIntegration(t *testing.T) {
	f := integrationFixture(t)
	e := activeLabEpisode(t, f)
	r := newLabRequest(t, f, e)
	approveLab(t, f, r.ID)
	processLab(t, f, r.ID)
	result := submitLab(t, f, r.ID)
	prior := notificationPage(t, f, "patient", "/api/v1/notifications")
	server := httptest.NewServer(f.api)
	t.Cleanup(server.Close)
	response, frames := connectStream(t, f, server.URL+"/api/v1/notifications/stream", "patient", fmt.Sprint(prior.Notifications[0].ID))
	if response.StatusCode != 200 {
		t.Fatal("could not open correction stream")
	}
	expectStatus(t, f.request("lab", "POST", "/api/v1/lab-results/"+result.ID+"/corrections", `{"reason":"Instrument corrected","value":"PRIVATE-CORRECTED","flag":"high"}`, true), 201)
	frame := nextFrame(t, frames)
	if frame.Event != "notification" || !strings.Contains(frame.Data, "lab_result.correct") || strings.Contains(frame.Data, "PRIVATE-CORRECTED") {
		t.Fatal("invalid live correction event", frame)
	}
	response.Body.Close()
	for _, actor := range []string{"patient", "doctor"} {
		page := notificationPage(t, f, actor, "/api/v1/notifications")
		if len(page.Notifications) == 0 || page.Notifications[0].Action != "lab_result.correct" || page.Notifications[0].ResourceID != result.ID {
			t.Fatalf("missing correction notification for %s", actor)
		}
		data, _ := json.Marshal(page)
		if strings.Contains(string(data), "PRIVATE-CORRECTED") || strings.Contains(string(data), "Instrument corrected") {
			t.Fatal("clinical content in notification")
		}
	}
	page, err := f.service.Timeline(context.Background(), f.tokens["patient"], timeline.Options{Kind: "laboratory", Limit: 100})
	if err != nil || len(page.Events) == 0 || page.Events[0].Action != "lab_result.correct" {
		t.Fatalf("missing correction timeline: %+v %v", page, err)
	}
}

func TestLabReceiptRecoveryIntegration(t *testing.T) {
	f := integrationFixture(t)
	e := activeLabEpisode(t, f)
	r := newLabRequest(t, f, e)
	approveLab(t, f, r.ID)
	processLab(t, f, r.ID)
	result := submitLab(t, f, r.ID)
	for _, actor := range []string{"lab", "labColleague"} {
		response := f.request(actor, "GET", "/api/v1/lab-receipts?limit=1&offset=0", "", false)
		expectStatus(t, response, 200)
		if !strings.Contains(response.Body.String(), result.ID) || strings.Contains(response.Body.String(), "SYNTHETIC-RESULT-42") || strings.Contains(response.Body.String(), "demo-unit") {
			t.Fatal("invalid receipt projection", response.Body.String())
		}
	}
	response := f.request("labOther", "GET", "/api/v1/lab-receipts", "", false)
	expectStatus(t, response, 200)
	if strings.Contains(response.Body.String(), result.ID) {
		t.Fatal("receipt crossed organization")
	}
	for _, actor := range []string{"patient", "doctor", "admin"} {
		expectStatus(t, f.request(actor, "GET", "/api/v1/lab-receipts", "", false), 403)
	}
	expectStatus(t, f.request("lab", "GET", "/api/v1/lab-receipts?offset=-1", "", false), 400)
	expectStatus(t, f.request("lab", "GET", "/api/v1/lab-results/"+result.ID, "", false), 403)
}

func TestWorkflowCancellationIntegration(t *testing.T) {
	for _, kind := range []string{"lab-requests", "prescriptions"} {
		for _, actor := range []string{"patient", "doctor", "provider"} {
			t.Run(kind+"/"+actor, func(t *testing.T) {
				f := integrationFixture(t)
				e := activeLabEpisode(t, f)
				id, provider := "", "lab"
				if kind == "lab-requests" {
					r := newLabRequest(t, f, e)
					id = r.ID
					approveLab(t, f, id)
				} else {
					p := newPrescription(t, f, e)
					id = p.ID
					provider = "pharmacy"
					approvePrescription(t, f, id)
				}
				who := actor
				if who == "provider" {
					who = provider
				}
				path := "/api/v1/" + kind + "/" + id
				for _, outsider := range []string{"other", "admin", "labOther", "pharmacyOther"} {
					expectStatus(t, f.request(outsider, "POST", path+"/cancel", `{"reason":"Not available"}`, true), 403)
				}
				expectStatus(t, f.request(who, "POST", path+"/cancel", `{"reason":"Not available"}`, false), 403)
				expectStatus(t, f.request(who, "POST", path+"/cancel", `{"reason":"no"}`, true), 400)
				expectStatus(t, f.request(who, "POST", path+"/cancel", `{"reason":"Not available"}`, true), 200)
				expectStatus(t, f.request(who, "POST", path+"/cancel", `{"reason":"Not available"}`, true), 409)
				expectStatus(t, f.request(provider, "PATCH", path+"/status", `{"status":"accepted"}`, true), kindStatus(kind))
				response := f.request("patient", "GET", path, "", false)
				expectStatus(t, response, 200)
				if !strings.Contains(response.Body.String(), `"status":"cancelled"`) || !strings.Contains(response.Body.String(), `"cancellation_note":"Not available"`) || strings.Contains(response.Body.String(), `"approved_at":null`) {
					t.Fatal("cancellation lost history", response.Body.String())
				}
				expectStatus(t, f.request(provider, "GET", path, "", false), 403)
				// Other assigned staff must hear that the item disappeared from their queue.
				if who != provider {
					notifications := notificationPage(t, f, provider, "/api/v1/notifications")
					if len(notifications.Notifications) == 0 || notifications.Notifications[0].Action != map[string]string{"lab-requests": "lab_request.cancel", "prescriptions": "prescription.cancel"}[kind] {
						t.Fatal("provider missing cancellation notification")
					}
				}
			})
		}
	}
}

func kindStatus(kind string) int {
	if kind == "lab-requests" {
		return 403
	}
	return 409
}

func TestCancellationTerminalRollbackAndRacesIntegration(t *testing.T) {
	for _, kind := range []string{"lab-requests", "prescriptions"} {
		t.Run(kind, func(t *testing.T) {
			f := integrationFixture(t)
			ctx := context.Background()
			e := activeLabEpisode(t, f)
			newItem := func() (string, string) {
				if kind == "lab-requests" {
					r := newLabRequest(t, f, e)
					return r.ID, "lab"
				}
				p := newPrescription(t, f, e)
				return p.ID, "pharmacy"
			}
			id, provider := newItem()
			path := "/api/v1/" + kind + "/" + id
			expectStatus(t, f.request(provider, "POST", path+"/cancel", `{"reason":"Provider unavailable"}`, true), 403)
			if kind == "lab-requests" {
				approveLab(t, f, id)
				processLab(t, f, id)
			} else {
				approvePrescription(t, f, id)
				for _, status := range []string{"accepted", "preparing", "ready"} {
					expectStatus(t, f.request(provider, "PATCH", path+"/status", `{"status":"`+status+`"}`, true), 200)
				}
			}
			// Both operations lock the same episode/resource; exactly one may win.
			start := make(chan struct{})
			done := make(chan error, 2)
			go func() {
				<-start
				if kind == "lab-requests" {
					done <- f.labService.Cancel(ctx, f.tokens["patient"], id, "Changed care plan")
				} else {
					done <- f.prescriptionService.Cancel(ctx, f.tokens["patient"], id, "Changed care plan")
				}
			}()
			go func() {
				<-start
				if kind == "lab-requests" {
					_, err := f.labService.SubmitResult(ctx, f.tokens[provider], id, laboratory.ResultInput{Value: "Synthetic", Flag: "normal"})
					done <- err
				} else {
					_, err := f.prescriptionService.UpdateStatus(ctx, f.tokens[provider], id, "completed")
					done <- err
				}
			}()
			close(start)
			a, b := <-done, <-done
			if (a == nil) == (b == nil) {
				t.Fatalf("expected one race winner: %v %v", a, b)
			}
			expectStatus(t, f.request("patient", "POST", path+"/cancel", `{"reason":"No longer needed"}`, true), 409)
			// Infrastructure failure must roll back status and notification together.
			id, provider = newItem()
			path = "/api/v1/" + kind + "/" + id
			if kind == "lab-requests" {
				approveLab(t, f, id)
			} else {
				approvePrescription(t, f, id)
			}
			if _, err := f.pool.Exec(ctx, `CREATE TRIGGER workflow_audit_failure BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION reject_audit_mutation()`); err != nil {
				t.Fatal(err)
			}
			expectStatus(t, f.request(provider, "POST", path+"/cancel", `{"reason":"Provider unavailable"}`, true), 503)
			var state string
			table := map[string]string{"lab-requests": "lab_requests", "prescriptions": "prescriptions"}[kind]
			if err := f.pool.QueryRow(ctx, "SELECT status FROM "+table+" WHERE id=$1", id).Scan(&state); err != nil || state != "sent" {
				t.Fatalf("cancellation did not rollback: %s %v", state, err)
			}
		})
	}
}
