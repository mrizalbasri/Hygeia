package care

import (
	"context"
	"testing"
)

func TestCarePassReadAfterAuditWait(t *testing.T) {
	for _, resource := range []string{"lab_result", "prescription"} {
		for _, expired := range []bool{false, true} {
			label := "valid"
			if expired {
				label = "expired"
			}
			t.Run(resource+"/"+label, func(t *testing.T) {
				f := integrationFixture(t)
				e := activeLabEpisode(t, f)
				path, id := "", ""
				if resource == "lab_result" {
					r := newLabRequest(t, f, e)
					approveLab(t, f, r.ID)
					processLab(t, f, r.ID)
					result := submitLab(t, f, r.ID)
					id = result.ID
					path = "/api/v1/lab-results/" + id
				} else {
					r := newPrescription(t, f, e)
					id = r.ID
					path = "/api/v1/prescriptions/" + id
				}
				c := createPass(t, f, resource, id)
				if _, err := f.carePassService.Redeem(context.Background(), f.tokens["colleague"], c.Token); err != nil {
					t.Fatal(err)
				}
				assertGrantAfterAuditWait(t, f, path, c.Pass.ExpiresAt, expired)
			})
		}
	}
}
