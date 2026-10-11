package care

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mrizalbasri/Hygeia/backend/internal/laboratory"
)

const resultJSON = `{"value":"SYNTHETIC-RESULT-42","unit":"demo-unit","reference_range":"demo-only","flag":"normal","note":"Synthetic test, not medical data"}`

func activeLabEpisode(t *testing.T, f *fixture) Episode {
	t.Helper()
	e, err := f.service.Invite(context.Background(), f.tokens["doctor"], "TEST-patient")
	if err != nil {
		t.Fatal(err)
	}
	e, err = f.service.Transition(context.Background(), f.tokens["patient"], e.ID, "accept")
	if err != nil {
		t.Fatal(err)
	}
	return e
}
func newLabRequest(t *testing.T, f *fixture, e Episode) laboratory.Request {
	t.Helper()
	body, _ := json.Marshal(laboratory.CreateInput{EpisodeID: e.ID, LabOrgID: f.ids["labOrg"], TestCode: "DEMO-TEST", TestName: "Synthetic lab test"})
	r := f.request("doctor", "POST", "/api/v1/lab-requests", string(body), true)
	expectStatus(t, r, 201)
	return labResponse(t, r)
}
func labResponse(t *testing.T, r *httptest.ResponseRecorder) laboratory.Request {
	t.Helper()
	var body struct {
		Request laboratory.Request `json:"lab_request"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Request
}
func approveLab(t *testing.T, f *fixture, id string) {
	t.Helper()
	expectStatus(t, f.request("patient", "POST", "/api/v1/lab-requests/"+id+"/approval", `{"decision":"approve"}`, true), 200)
}
func processLab(t *testing.T, f *fixture, id string) {
	t.Helper()
	for _, status := range []string{"accepted", "in_progress"} {
		expectStatus(t, f.request("lab", "PATCH", "/api/v1/lab-requests/"+id+"/status", `{"status":"`+status+`"}`, true), 200)
	}
}
func submitLab(t *testing.T, f *fixture, id string) laboratory.Receipt {
	t.Helper()
	r := f.request("labColleague", "POST", "/api/v1/lab-requests/"+id+"/result", resultJSON, true)
	expectStatus(t, r, 201)
	if strings.Contains(r.Body.String(), "SYNTHETIC-RESULT-42") {
		t.Fatal("submission receipt includes result payload")
	}
	var body struct {
		Result laboratory.Receipt `json:"result"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Result.Status != "completed" || body.Result.LabRequestID != id || body.Result.ID == "" {
		t.Fatal("invalid result receipt")
	}
	return body.Result
}
func assertLabProjection(t *testing.T, r *httptest.ResponseRecorder) {
	t.Helper()
	var body struct {
		Request map[string]any `json:"lab_request"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Request) != 5 {
		t.Fatalf("lab response has extra fields: %s", r.Body.String())
	}
	for _, key := range []string{"id", "patient_ref", "test_code", "test_name", "status"} {
		if _, ok := body.Request[key]; !ok {
			t.Fatalf("missing lab field %s", key)
		}
	}
}

func TestLaboratoryLifecycleIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	expectStatus(t, f.request("", "GET", "/api/v1/lab-requests", "", false), 401)
	labs := f.request("doctor", "GET", "/api/v1/laboratories", "", false)
	expectStatus(t, labs, 200)
	var providers struct {
		Labs []map[string]any `json:"laboratories"`
	}
	if err := json.Unmarshal(labs.Body.Bytes(), &providers); err != nil || len(providers.Labs) != 2 {
		t.Fatal("laboratory directory incorrect")
	}
	for _, lab := range providers.Labs {
		if len(lab) != 2 || lab["id"] == nil || lab["name"] == nil {
			t.Fatal("directory leaks metadata")
		}
	}
	for _, actor := range []string{"patient", "lab", "admin"} {
		expectStatus(t, f.request(actor, "GET", "/api/v1/laboratories", "", false), 403)
	}
	e, err := f.service.Invite(ctx, f.tokens["doctor"], "TEST-patient")
	if err != nil {
		t.Fatal(err)
	}
	pendingCreate, _ := json.Marshal(laboratory.CreateInput{EpisodeID: e.ID, LabOrgID: f.ids["labOrg"], TestCode: "DEMO", TestName: "Synthetic test"})
	expectStatus(t, f.request("doctor", "POST", "/api/v1/lab-requests", string(pendingCreate), true), 403)
	e, err = f.service.Transition(ctx, f.tokens["patient"], e.ID, "accept")
	if err != nil {
		t.Fatal(err)
	}
	createBody, _ := json.Marshal(laboratory.CreateInput{EpisodeID: e.ID, LabOrgID: f.ids["labOrg"], TestCode: "DEMO", TestName: "Synthetic test"})
	expectStatus(t, f.request("doctor", "POST", "/api/v1/lab-requests", string(createBody), false), 403)
	for _, actor := range []string{"patient", "colleague", "lab", "pharmacy", "admin"} {
		expectStatus(t, f.request(actor, "POST", "/api/v1/lab-requests", string(createBody), true), 403)
	}
	badProvider, _ := json.Marshal(laboratory.CreateInput{EpisodeID: e.ID, LabOrgID: f.ids["doctorOrg"], TestCode: "DEMO", TestName: "Synthetic test"})
	expectStatus(t, f.request("doctor", "POST", "/api/v1/lab-requests", string(badProvider), true), 403)
	request := newLabRequest(t, f, e)
	path := "/api/v1/lab-requests/" + request.ID
	if request.Status != "pending_patient_approval" || request.ApprovedAt != nil || request.ResultID != nil {
		t.Fatal("invalid initial request")
	}
	for _, actor := range []string{"other", "colleague", "lab", "labOther", "pharmacy", "admin"} {
		expectStatus(t, f.request(actor, "GET", path, "", false), 403)
	}
	pendingList := f.request("lab", "GET", "/api/v1/lab-requests", "", false)
	expectStatus(t, pendingList, 200)
	if !strings.Contains(pendingList.Body.String(), `"lab_requests":[]`) {
		t.Fatal("lab sees unapproved request")
	}
	expectStatus(t, f.request("lab", "PATCH", path+"/status", `{"status":"accepted"}`, true), 403)
	expectStatus(t, f.request("lab", "POST", path+"/result", resultJSON, true), 403)
	for _, actor := range []string{"other", "doctor", "lab", "admin"} {
		expectStatus(t, f.request(actor, "POST", path+"/approval", `{"decision":"approve"}`, true), 403)
	}
	expectStatus(t, f.request("patient", "POST", path+"/approval", `{"decision":"approve"}`, false), 403)
	approveLab(t, f, request.ID)
	expectStatus(t, f.request("patient", "POST", path+"/approval", `{"decision":"approve"}`, true), 409)
	for _, actor := range []string{"patient", "doctor"} {
		expectStatus(t, f.request(actor, "GET", path, "", false), 200)
	}
	expectStatus(t, f.request("labOther", "GET", path, "", false), 403)
	for _, actor := range []string{"lab", "labColleague"} {
		view := f.request(actor, "GET", path, "", false)
		expectStatus(t, view, 200)
		assertLabProjection(t, view)
		list := f.request(actor, "GET", "/api/v1/lab-requests?status=sent&limit=1", "", false)
		expectStatus(t, list, 200)
		if !strings.Contains(list.Body.String(), request.ID) || strings.Contains(list.Body.String(), "episode_id") {
			t.Fatal("queue contents or redaction incorrect")
		}
	}
	expectStatus(t, f.request("labOther", "PATCH", path+"/status", `{"status":"accepted"}`, true), 403)
	expectStatus(t, f.request("lab", "PATCH", path+"/status", `{"status":"in_progress"}`, true), 409)
	expectStatus(t, f.request("lab", "POST", path+"/result", resultJSON, true), 409)
	processLab(t, f, request.ID)
	expectStatus(t, f.request("lab", "POST", path+"/result", resultJSON, false), 403)
	receipt := submitLab(t, f, request.ID)
	resultPath := "/api/v1/lab-results/" + receipt.ID
	expectStatus(t, f.request("lab", "POST", path+"/result", resultJSON, true), 409)
	expectStatus(t, f.request("lab", "GET", path, "", false), 403)
	completedQueue := f.request("lab", "GET", "/api/v1/lab-requests?status=completed", "", false)
	expectStatus(t, completedQueue, 200)
	if !strings.Contains(completedQueue.Body.String(), `"lab_requests":[]`) {
		t.Fatal("completed request remains in lab queue")
	}
	for _, actor := range []string{"patient", "doctor"} {
		result := f.request(actor, "GET", resultPath, "", false)
		expectStatus(t, result, 200)
		var body struct {
			Result laboratory.Result `json:"result"`
		}
		if err := json.Unmarshal(result.Body.Bytes(), &body); err != nil || body.Result.Value != "SYNTHETIC-RESULT-42" || body.Result.TestName != request.TestName {
			t.Fatal("incorrect structured result")
		}
		detail := f.request(actor, "GET", path, "", false)
		expectStatus(t, detail, 200)
		if labResponse(t, detail).ResultID == nil || strings.Contains(detail.Body.String(), "SYNTHETIC-RESULT-42") {
			t.Fatal("request embeds clinical payload or omits result ID")
		}
	}
	for _, actor := range []string{"other", "colleague", "lab", "labColleague", "labOther", "pharmacy", "admin"} {
		result := f.request(actor, "GET", resultPath, "", false)
		expectStatus(t, result, 403)
		if strings.Contains(result.Body.String(), "SYNTHETIC-RESULT-42") || strings.Contains(result.Body.String(), request.TestName) {
			t.Fatal("denied result leaked clinical data")
		}
	}
	if _, err := f.pool.Exec(ctx, `UPDATE organization_memberships SET organization_id=$2 WHERE user_id=$1`, f.ids["doctor"], f.ids["labOrg"]); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("doctor", "GET", resultPath, "", false), 401)
	if _, err := f.pool.Exec(ctx, `UPDATE organization_memberships SET organization_id=$2 WHERE user_id=$1`, f.ids["doctor"], f.ids["doctorOrg"]); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("patient", "GET", "/api/v1/lab-results/00000000-0000-0000-0000-000000000001", "", false), 403)
	var leaked int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE context::text LIKE '%SYNTHETIC-RESULT-42%'`).Scan(&leaked); err != nil || leaked != 0 {
		t.Fatal("result value entered audit")
	}
	if _, err := f.pool.Exec(ctx, `UPDATE lab_results SET value='tampered' WHERE id=$1`, receipt.ID); err == nil {
		t.Fatal("result mutation allowed")
	}
	if _, err := f.pool.Exec(ctx, `DELETE FROM lab_results WHERE id=$1`, receipt.ID); err == nil {
		t.Fatal("result deletion allowed")
	}
	verify, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verify.Intact {
		t.Fatalf("audit verification: %+v %v", verify, err)
	}
	summary := f.request("patient", "GET", "/api/v1/patient/access-summary?limit=100", "", false)
	expectStatus(t, summary, 200)
	if !strings.Contains(summary.Body.String(), "hasil laboratorium") {
		t.Fatal("missing lab audit summary")
	}
	if _, err := f.pool.Exec(ctx, `UPDATE users SET disabled_at=now() WHERE id=$1`, f.ids["doctor"]); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("doctor", "GET", resultPath, "", false), 401)
	if _, err := f.pool.Exec(ctx, `UPDATE users SET disabled_at=NULL WHERE id=$1`, f.ids["doctor"]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE sessions SET created_at=now()-interval '2 hours',expires_at=now()-interval '1 hour' WHERE user_id=$1`, f.ids["lab"]); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("lab", "GET", "/api/v1/lab-requests", "", false), 401)
}

func TestLaboratoryEpisodeEndIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e := activeLabEpisode(t, f)
	pending := newLabRequest(t, f, e)
	sent := newLabRequest(t, f, e)
	rejected := newLabRequest(t, f, e)
	approveLab(t, f, sent.ID)
	expectStatus(t, f.request("patient", "POST", "/api/v1/lab-requests/"+rejected.ID+"/approval", `{"decision":"reject"}`, true), 200)
	if _, err := f.service.Transition(ctx, f.tokens["doctor"], e.ID, "end"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ id, reason string }{{pending.ID, "episode_ended"}, {rejected.ID, "patient_rejected"}} {
		view := f.request("patient", "GET", "/api/v1/lab-requests/"+tc.id, "", false)
		expectStatus(t, view, 200)
		r := labResponse(t, view)
		if r.Status != "cancelled" || r.CancellationReason == nil || *r.CancellationReason != tc.reason || r.CancelledAt == nil {
			t.Fatal("incorrect cancellation state")
		}
		expectStatus(t, f.request("patient", "POST", "/api/v1/lab-requests/"+tc.id+"/approval", `{"decision":"approve"}`, true), 409)
		expectStatus(t, f.request("lab", "GET", "/api/v1/lab-requests/"+tc.id, "", false), 403)
	}
	expectStatus(t, f.request("doctor", "GET", "/api/v1/lab-requests/"+sent.ID, "", false), 403)
	afterEndList := f.request("doctor", "GET", "/api/v1/lab-requests", "", false)
	expectStatus(t, afterEndList, 200)
	if !strings.Contains(afterEndList.Body.String(), `"lab_requests":[]`) {
		t.Fatal("doctor retains ended episode lab access")
	}
	processLab(t, f, sent.ID)
	receipt := submitLab(t, f, sent.ID)
	expectStatus(t, f.request("patient", "GET", "/api/v1/lab-results/"+receipt.ID, "", false), 200)
	expectStatus(t, f.request("doctor", "GET", "/api/v1/lab-results/"+receipt.ID, "", false), 403)
	body, _ := json.Marshal(laboratory.CreateInput{EpisodeID: e.ID, LabOrgID: f.ids["labOrg"], TestCode: "NEW", TestName: "New test"})
	expectStatus(t, f.request("doctor", "POST", "/api/v1/lab-requests", string(body), true), 403)
	verify, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verify.Intact {
		t.Fatal("audit broken after episode end")
	}
}

func TestLaboratoryConcurrencyIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		e := activeLabEpisode(t, f)
		request := newLabRequest(t, f, e)
		start := make(chan struct{})
		decisions := make(chan error, 1)
		endings := make(chan error, 1)
		go func() {
			<-start
			_, err := f.labService.Approve(ctx, f.tokens["patient"], request.ID, "approve")
			decisions <- err
		}()
		go func() { <-start; _, err := f.service.Transition(ctx, f.tokens["patient"], e.ID, "end"); endings <- err }()
		close(start)
		decision, end := <-decisions, <-endings
		if end != nil {
			t.Fatal(end)
		}
		if decision != nil && !errors.Is(decision, laboratory.ErrConflict) {
			t.Fatal(decision)
		}
		var status string
		var approved *time.Time
		if err := f.pool.QueryRow(ctx, `SELECT status,approved_at FROM lab_requests WHERE id=$1`, request.ID).Scan(&status, &approved); err != nil {
			t.Fatal(err)
		}
		if decision == nil && (status != "sent" || approved == nil) || decision != nil && (status != "cancelled" || approved != nil) {
			t.Fatal("approval/end race produced contradictory state")
		}
	}
	e := activeLabEpisode(t, f)
	request := newLabRequest(t, f, e)
	approveLab(t, f, request.ID)
	var group sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := f.labService.UpdateStatus(ctx, f.tokens["lab"], request.ID, "accepted")
			results <- err
		}()
	}
	group.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, laboratory.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatal("concurrent lab transitions both succeeded")
	}
	if _, err := f.labService.UpdateStatus(ctx, f.tokens["lab"], request.ID, "in_progress"); err != nil {
		t.Fatal(err)
	}
	var input laboratory.ResultInput
	if err := json.Unmarshal([]byte(resultJSON), &input); err != nil {
		t.Fatal(err)
	}
	results = make(chan error, 2)
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := f.labService.SubmitResult(ctx, f.tokens["lab"], request.ID, input)
			results <- err
		}()
	}
	group.Wait()
	close(results)
	successes, conflicts = 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, laboratory.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatal("duplicate concurrent result accepted")
	}
	verify, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verify.Intact {
		t.Fatal("concurrent lab audit broken")
	}
}

func TestLaboratoryAuditFailureIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e := activeLabEpisode(t, f)
	pending := newLabRequest(t, f, e)
	processing := newLabRequest(t, f, e)
	sent := newLabRequest(t, f, e)
	approveLab(t, f, sent.ID)
	approveLab(t, f, processing.ID)
	processLab(t, f, processing.ID)
	if _, err := f.pool.Exec(ctx, `CREATE FUNCTION fail_lab_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test audit failure'; END $$;
        CREATE TRIGGER fail_lab_audit BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION fail_lab_audit()`); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(laboratory.CreateInput{EpisodeID: e.ID, LabOrgID: f.ids["labOrg"], TestCode: "FAIL", TestName: "Synthetic failure"})
	expectStatus(t, f.request("doctor", "POST", "/api/v1/lab-requests", string(body), true), 503)
	expectStatus(t, f.request("patient", "POST", "/api/v1/lab-requests/"+pending.ID+"/approval", `{"decision":"approve"}`, true), 503)
	expectStatus(t, f.request("lab", "PATCH", "/api/v1/lab-requests/"+sent.ID+"/status", `{"status":"accepted"}`, true), 503)
	expectStatus(t, f.request("lab", "POST", "/api/v1/lab-requests/"+processing.ID+"/result", resultJSON, true), 503)
	expectStatus(t, f.request("patient", "PATCH", "/api/v1/episodes/"+e.ID, `{"action":"end"}`, true), 503)
	var episodeStatus, pendingStatus, processStatus, sentStatus string
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT status FROM care_episodes WHERE id=$1`, e.ID).Scan(&episodeStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT status FROM lab_requests WHERE id=$1`, pending.ID).Scan(&pendingStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT status FROM lab_requests WHERE id=$1`, processing.ID).Scan(&processStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT status FROM lab_requests WHERE id=$1`, sent.ID).Scan(&sentStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM lab_results)+(SELECT count(*) FROM lab_requests WHERE test_code='FAIL')`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if episodeStatus != "active" || pendingStatus != "pending_patient_approval" || processStatus != "in_progress" || sentStatus != "sent" || count != 0 {
		t.Fatal("business data survived failed audit transaction")
	}
	if _, err := f.pool.Exec(ctx, `DROP TRIGGER fail_lab_audit ON audit_events; DROP FUNCTION fail_lab_audit()`); err != nil {
		t.Fatal(err)
	}
	receipt := submitLab(t, f, processing.ID)
	if _, err := f.pool.Exec(ctx, `CREATE TRIGGER fail_lab_audit BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION reject_audit_mutation()`); err != nil {
		t.Fatal(err)
	}
	hidden := f.request("patient", "GET", "/api/v1/lab-results/"+receipt.ID, "", false)
	expectStatus(t, hidden, 503)
	if strings.Contains(hidden.Body.String(), "SYNTHETIC-RESULT-42") {
		t.Fatal("result returned when audit failed")
	}
	if _, err := f.pool.Exec(ctx, `DROP TRIGGER fail_lab_audit ON audit_events`); err != nil {
		t.Fatal(err)
	}
	verify, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verify.Intact {
		t.Fatal("failed transaction damaged audit")
	}
}
