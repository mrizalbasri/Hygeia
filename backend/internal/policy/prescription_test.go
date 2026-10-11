package policy

import (
	"testing"

	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

func TestPrescriptionAccessMatrix(t *testing.T) {
	clinic := &auth.Organization{ID: "clinic", Type: "clinic"}
	pharmacy := &auth.Organization{ID: "pharmacy", Type: "pharmacy"}
	cases := []struct {
		name                      string
		actor                     auth.User
		patient, doctor, assigned bool
	}{
		{"patient", auth.User{ID: "patient", Role: "patient"}, true, false, false},
		{"other patient", auth.User{ID: "other", Role: "patient"}, false, false, false},
		{"doctor", auth.User{ID: "doctor", Role: "doctor", Organization: clinic}, false, true, false},
		{"colleague", auth.User{ID: "colleague", Role: "doctor", Organization: clinic}, false, false, false},
		{"transferred doctor", auth.User{ID: "doctor", Role: "doctor", Organization: &auth.Organization{ID: "other", Type: "clinic"}}, false, false, false},
		{"pharmacy", auth.User{ID: "staff", Role: "pharmacy", Organization: pharmacy}, false, false, true},
		{"pharmacy colleague", auth.User{ID: "colleague", Role: "pharmacy", Organization: pharmacy}, false, false, true},
		{"other pharmacy", auth.User{ID: "other", Role: "pharmacy", Organization: &auth.Organization{ID: "other", Type: "pharmacy"}}, false, false, false},
		{"wrong org type", auth.User{ID: "staff", Role: "pharmacy", Organization: &auth.Organization{ID: "pharmacy", Type: "clinic"}}, false, false, false},
		{"pharmacy without org", auth.User{ID: "staff", Role: "pharmacy"}, false, false, false},
		{"lab", auth.User{Role: "lab", Organization: pharmacy}, false, false, false},
		{"admin", auth.User{Role: "admin"}, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, episodeStatus := range []string{"pending", "active", "rejected", "ended"} {
				for _, status := range []string{"pending_patient_approval", "sent", "accepted", "preparing", "ready", "completed", "cancelled"} {
					p := Prescription{Episode: Episode{DoctorID: "doctor", PatientUserID: "patient", OrganizationID: "clinic", Status: episodeStatus}, PharmacyOrgID: "pharmacy", Status: status}
					processing := status == "sent" || status == "accepted" || status == "preparing" || status == "ready"
					if CanViewPrescription(tc.actor, p) != (tc.patient || tc.doctor && episodeStatus == "active" || tc.assigned && processing) {
						t.Fatalf("view %s/%s", episodeStatus, status)
					}
					if CanProcessPrescription(tc.actor, p) != (tc.assigned && processing) {
						t.Fatalf("process %s/%s", episodeStatus, status)
					}
					if CanDecidePrescription(tc.actor, p) != tc.patient {
						t.Fatal("approval ownership")
					}
				}
			}
		})
	}
}

func TestPrescriptionTransitions(t *testing.T) {
	next := map[string]string{"sent": "accepted", "accepted": "preparing", "preparing": "ready", "ready": "completed"}
	for _, from := range []string{"pending_patient_approval", "sent", "accepted", "preparing", "ready", "completed", "cancelled", "invalid"} {
		for _, to := range []string{"pending_patient_approval", "sent", "accepted", "preparing", "ready", "completed", "cancelled", "invalid"} {
			if CanTransitionPrescription(from, to) != (next[from] == to) {
				t.Fatalf("transition %s -> %s", from, to)
			}
		}
	}
}
