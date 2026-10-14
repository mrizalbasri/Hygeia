package care

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

func TestAdminManagementIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	for _, path := range []string{"/admin/users", "/admin/organizations", "/admin/audit-chains", "/admin/activity"} {
		expectStatus(t, f.request("", "GET", "/api/v1"+path, "", false), 401)
		for _, actor := range []string{"patient", "doctor", "lab", "pharmacy"} {
			expectStatus(t, f.request(actor, "GET", "/api/v1"+path, "", false), 403)
		}
		expectStatus(t, f.request("admin", "GET", "/api/v1"+path, "", false), 200)
	}
	body := `{"name":"Synthetic new clinic","type":"clinic"}`
	expectStatus(t, f.request("admin", "POST", "/api/v1/admin/organizations", body, false), 403)
	r := f.request("admin", "POST", "/api/v1/admin/organizations", body, true)
	expectStatus(t, r, 201)
	var org struct {
		Organization auth.Organization `json:"organization"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &org); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("admin", "POST", "/api/v1/admin/organizations", body, true), 409)
	expectStatus(t, f.request("admin", "PATCH", "/api/v1/admin/organizations/"+org.Organization.ID, `{"name":"Renamed synthetic clinic"}`, true), 200)
	create := `{"email":"new-doctor@demo.test","full_name":"Synthetic new doctor","password":"synthetic-long-password","role":"doctor","organization_id":"` + org.Organization.ID + `"}`
	r = f.request("admin", "POST", "/api/v1/admin/users", create, true)
	expectStatus(t, r, 201)
	if strings.Contains(r.Body.String(), "password") || strings.Contains(r.Body.String(), "argon2") {
		t.Fatal("create leaked password")
	}
	var account struct {
		User struct {
			ID         string  `json:"id"`
			PatientRef *string `json:"patient_ref"`
		} `json:"user"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &account); err != nil {
		t.Fatal(err)
	}
	var hash string
	if err := f.pool.QueryRow(ctx, `SELECT password_hash FROM users WHERE id=$1`, account.User.ID).Scan(&hash); err != nil || !auth.VerifyPassword("synthetic-long-password", hash) {
		t.Fatal("password not hashed correctly")
	}
	expectStatus(t, f.request("admin", "POST", "/api/v1/admin/users", create, true), 409)
	expectStatus(t, f.request("admin", "POST", "/api/v1/admin/users", strings.Replace(create, org.Organization.ID, f.ids["labOrg"], 1), true), 400)
	r = f.request("admin", "POST", "/api/v1/admin/users", `{"email":"new-patient@demo.test","full_name":"Synthetic patient","password":"synthetic-long-password","role":"patient"}`, true)
	expectStatus(t, r, 201)
	if err := json.Unmarshal(r.Body.Bytes(), &account); err != nil || account.User.PatientRef == nil || !strings.HasPrefix(*account.User.PatientRef, "HYG-") {
		t.Fatal("patient reference missing")
	}
	expectStatus(t, f.request("admin", "PATCH", "/api/v1/admin/users/"+f.ids["admin"], `{"disabled":true}`, true), 409)
	expectStatus(t, f.request("admin", "PATCH", "/api/v1/admin/users/"+f.ids["doctor"], `{"role":"admin"}`, true), 400)
	expectStatus(t, f.request("admin", "PATCH", "/api/v1/admin/users/"+f.ids["doctor"], `{"disabled":true}`, true), 200)
	expectStatus(t, f.request("doctor", "GET", "/api/v1/me", "", false), 401)
	expectStatus(t, f.request("admin", "PATCH", "/api/v1/admin/users/"+f.ids["doctor"], `{"disabled":false}`, true), 200)
	expectStatus(t, f.request("doctor", "GET", "/api/v1/me", "", false), 401)
	r = f.request("admin", "GET", "/api/v1/admin/activity", "", false)
	expectStatus(t, r, 200)
	if strings.Contains(r.Body.String(), "synthetic-long-password") || strings.Contains(r.Body.String(), "password_hash") {
		t.Fatal("activity leaked credentials")
	}
	if _, err := f.pool.Exec(ctx, `UPDATE admin_activity SET action='user.create'`); err == nil {
		t.Fatal("activity update accepted")
	}
}
func TestAdminAtomicityIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `CREATE FUNCTION fail_admin_activity() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic activity failure'; END $$; CREATE TRIGGER fail_admin_activity BEFORE INSERT ON admin_activity FOR EACH ROW EXECUTE FUNCTION fail_admin_activity()`); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("admin", "PATCH", "/api/v1/admin/users/"+f.ids["doctor"], `{"disabled":true}`, true), 503)
	expectStatus(t, f.request("doctor", "GET", "/api/v1/me", "", false), 200)
	expectStatus(t, f.request("admin", "POST", "/api/v1/admin/users", `{"email":"rollback@demo.test","full_name":"Synthetic","password":"synthetic-long-password","role":"patient"}`, true), 503)
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE email='rollback@demo.test'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("user survived failed activity write")
	}
}
