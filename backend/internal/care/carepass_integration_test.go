package care

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/mrizalbasri/Hygeia/backend/internal/carepass"
	"image/png"
	"strings"
	"sync"
	"testing"
	"time"
)

func createPass(t *testing.T, f *fixture, kind, id string) carepass.Created {
	t.Helper()
	r := f.request("patient", "POST", "/api/v1/carepasses", `{"resource_type":"`+kind+`","resource_id":"`+id+`","ttl_minutes":15}`, true)
	expectStatus(t, r, 201)
	var c carepass.Created
	if err := json.Unmarshal(r.Body.Bytes(), &c); err != nil {
		t.Fatal(err)
	}
	return c
}
func TestCarePassScopedTokenRevokeAndExpiry(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e := activeLabEpisode(t, f)
	r := newLabRequest(t, f, e)
	approveLab(t, f, r.ID)
	processLab(t, f, r.ID)
	result := submitLab(t, f, r.ID)
	rx := newPrescription(t, f, e)
	c := createPass(t, f, "lab_result", result.ID)
	if c.Pass.ExpiresAt.Sub(c.Pass.CreatedAt) != 15*time.Minute || !strings.HasSuffix(c.ShareURL, "#token="+c.Token) {
		t.Fatal("invalid pass lifetime or URL")
	}
	pngBytes, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(c.QR, "data:image/png;base64,"))
	if err != nil {
		t.Fatal(err)
	}
	image, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil || image.Bounds().Dx() != 384 {
		t.Fatal("invalid QR PNG")
	}
	var stored []byte
	if err := f.pool.QueryRow(ctx, `SELECT token_hash FROM carepasses WHERE id=$1`, c.Pass.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(c.Token))
	if !bytes.Equal(stored, hash[:]) {
		t.Fatal("token not hashed")
	}
	redeem := `{"token":"` + c.Token + `"}`
	expectStatus(t, f.request("colleague", "POST", "/api/v1/carepasses/redeem", redeem, false), 403)
	for _, actor := range []string{"patient", "lab", "pharmacy", "admin"} {
		expectStatus(t, f.request(actor, "POST", "/api/v1/carepasses/redeem", redeem, true), 403)
	}
	expectStatus(t, f.request("colleague", "POST", "/api/v1/carepasses/redeem", redeem, true), 200)
	expectStatus(t, f.request("colleague", "POST", "/api/v1/carepasses/redeem", redeem, true), 200)
	expectStatus(t, f.request("doctor", "POST", "/api/v1/carepasses/redeem", redeem, true), 403)
	expectStatus(t, f.request("colleague", "GET", "/api/v1/lab-results/"+result.ID, "", false), 200)
	expectStatus(t, f.request("colleague", "GET", "/api/v1/prescriptions/"+rx.ID, "", false), 403)
	list := f.request("patient", "GET", "/api/v1/carepasses", "", false)
	expectStatus(t, list, 200)
	if strings.Contains(list.Body.String(), c.Token) || strings.Contains(list.Body.String(), "token_hash") {
		t.Fatal("list leaked token")
	}
	expectStatus(t, f.request("other", "PATCH", "/api/v1/carepasses/"+c.Pass.ID, `{"action":"revoke"}`, true), 403)
	expectStatus(t, f.request("patient", "PATCH", "/api/v1/carepasses/"+c.Pass.ID, `{"action":"revoke"}`, true), 200)
	expectStatus(t, f.request("colleague", "GET", "/api/v1/lab-results/"+result.ID, "", false), 403)
	expectStatus(t, f.request("colleague", "POST", "/api/v1/carepasses/redeem", redeem, true), 403)
	p := createPass(t, f, "prescription", rx.ID)
	expectStatus(t, f.request("colleague", "POST", "/api/v1/carepasses/redeem", `{"token":"`+p.Token+`"}`, true), 200)
	expectStatus(t, f.request("colleague", "GET", "/api/v1/prescriptions/"+rx.ID, "", false), 200)
	f.prescriptionService.Now = func() time.Time { return p.Pass.ExpiresAt }
	f.carePassService.Now = f.prescriptionService.Now
	expectStatus(t, f.request("colleague", "GET", "/api/v1/prescriptions/"+rx.ID, "", false), 403)
	expectStatus(t, f.request("colleague", "POST", "/api/v1/carepasses/redeem", `{"token":"`+p.Token+`"}`, true), 403)
	audit := f.request("patient", "GET", "/api/v1/patient/audit", "", false)
	if strings.Contains(audit.Body.String(), c.Token) {
		t.Fatal("audit leaked secret")
	}
}
func TestCarePassConcurrentRedemption(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e := activeLabEpisode(t, f)
	rx := newPrescription(t, f, e)
	c := createPass(t, f, "prescription", rx.ID)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, actor := range []string{"doctor", "colleague"} {
		wg.Add(1)
		go func(actor string) {
			defer wg.Done()
			_, err := f.carePassService.Redeem(ctx, f.tokens[actor], c.Token)
			errs <- err
		}(actor)
	}
	wg.Wait()
	close(errs)
	success := 0
	for err := range errs {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("%d successful redemptions", success)
	}
}
func TestCarePassRollbackAndOwnership(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e := activeLabEpisode(t, f)
	rx := newPrescription(t, f, e)
	input := `{"resource_type":"prescription","resource_id":"` + rx.ID + `","ttl_minutes":60}`
	expectStatus(t, f.request("other", "POST", "/api/v1/carepasses", input, true), 403)
	expectStatus(t, f.request("doctor", "POST", "/api/v1/carepasses", input, true), 403)
	c := createPass(t, f, "prescription", rx.ID)
	if _, err := f.pool.Exec(ctx, `CREATE FUNCTION fail_pass_notification() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic'; END $$; CREATE TRIGGER fail_pass_notification BEFORE INSERT ON notifications FOR EACH ROW EXECUTE FUNCTION fail_pass_notification()`); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("colleague", "POST", "/api/v1/carepasses/redeem", `{"token":"`+c.Token+`"}`, true), 503)
	var grantee *string
	if err := f.pool.QueryRow(ctx, `SELECT grantee_user_id::text FROM carepasses WHERE id=$1`, c.Pass.ID).Scan(&grantee); err != nil || grantee != nil {
		t.Fatal("redemption survived rollback")
	}
}
