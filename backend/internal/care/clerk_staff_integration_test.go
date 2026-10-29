package care

import (
	"context"
	"errors"
	"github.com/mrizalbasri/Hygeia/backend/internal/admin"
	"github.com/mrizalbasri/Hygeia/backend/internal/apiutil"
	"testing"
)

func TestClerkStaffAssignmentIntegration(t *testing.T) {
	for _, used := range []bool{false, true} {
		name := "pristine"
		if used {
			name = "clinical-history"
		}
		t.Run(name, func(t *testing.T) {
			f := integrationFixture(t)
			ctx := context.Background()
			svc := &admin.Service{DB: f.pool}
			if _, err := f.pool.Exec(ctx, `UPDATE users SET clerk_user_id='user_StaffOnboarding' WHERE id=$1`, f.ids["patient"]); err != nil {
				t.Fatal(err)
			}
			if used {
				activeLabEpisode(t, f)
			}
			if _, err := svc.AssignStaff(ctx, f.tokens["patient"], f.ids["patient"], "doctor", f.ids["doctorOrg"]); !errors.Is(err, apiutil.ErrForbidden) {
				t.Fatal("patient self-promoted", err)
			}
			account, err := svc.AssignStaff(ctx, f.tokens["admin"], f.ids["patient"], "doctor", f.ids["doctorOrg"])
			if used {
				if !errors.Is(err, apiutil.ErrConflict) {
					t.Fatal("clinical patient converted", err)
				}
				var count int
				if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM patients WHERE user_id=$1`, f.ids["patient"]).Scan(&count); err != nil || count != 1 {
					t.Fatal("clinical data lost")
				}
				return
			}
			if err != nil || account.Role != "doctor" || account.OrganizationID == nil || *account.OrganizationID != f.ids["doctorOrg"] {
				t.Fatalf("staff assignment failed %+v %v", account, err)
			}
			var count int
			if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM sessions WHERE user_id=$1`, f.ids["patient"]).Scan(&count); err != nil || count != 0 {
				t.Fatal("old sessions retained")
			}
			if _, err := svc.AssignStaff(ctx, f.tokens["admin"], f.ids["patient"], "admin", ""); !errors.Is(err, apiutil.ErrConflict) {
				t.Fatal("existing staff role silently changed", err)
			}
		})
	}
}
