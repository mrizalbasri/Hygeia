package policy

import (
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"testing"
)

func TestLaboratoryAccessMatrix(t *testing.T) {
	doctor := auth.User{ID: "doctor", Role: "doctor", Organization: &auth.Organization{ID: "clinic", Type: "clinic"}}
	patient := auth.User{ID: "patient", Role: "patient"}
	lab := auth.User{ID: "lab", Role: "lab", Organization: &auth.Organization{ID: "laboratory", Type: "laboratory"}}
	cases := []struct {
		name                            string
		actor                           auth.User
		ownPatient, ownDoctor, assigned bool
	}{
		{"patient", patient, true, false, false},
		{"other patient", auth.User{ID: "other", Role: "patient"}, false, false, false},
		{"inviting doctor", doctor, false, true, false},
		{"clinic colleague", auth.User{ID: "colleague", Role: "doctor", Organization: doctor.Organization}, false, false, false},
		{"transferred doctor", auth.User{ID: "doctor", Role: "doctor", Organization: &auth.Organization{ID: "elsewhere", Type: "clinic"}}, false, false, false},
		{"assigned lab", lab, false, false, true},
		{"lab colleague", auth.User{ID: "colleague", Role: "lab", Organization: lab.Organization}, false, false, true},
		{"other lab", auth.User{ID: "lab", Role: "lab", Organization: &auth.Organization{ID: "other", Type: "laboratory"}}, false, false, false},
		{"lab without org", auth.User{ID: "lab", Role: "lab"}, false, false, false},
		{"pharmacy", auth.User{ID: "pharmacy", Role: "pharmacy"}, false, false, false},
		{"admin", auth.User{ID: "admin", Role: "admin"}, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, episodeStatus := range []string{"pending", "active", "rejected", "ended"} {
				for _, status := range []string{"pending_patient_approval", "sent", "accepted", "in_progress", "completed", "cancelled"} {
					request := LabRequest{Episode: Episode{DoctorID: "doctor", PatientUserID: "patient", OrganizationID: "clinic", Status: episodeStatus}, LabOrgID: "laboratory", Status: status}
					processing := status == "sent" || status == "accepted" || status == "in_progress"
					activeDoctor := tc.ownDoctor && episodeStatus == "active"
					if CanViewLabRequest(tc.actor, request) != (tc.ownPatient || activeDoctor || tc.assigned && processing) {
						t.Fatalf("request access: %s/%s", episodeStatus, status)
					}
					if CanProcessLabRequest(tc.actor, request) != (tc.assigned && processing) {
						t.Fatalf("processing: %s/%s", episodeStatus, status)
					}
					if CanViewLabResult(tc.actor, request) != (status == "completed" && (tc.ownPatient || activeDoctor)) {
						t.Fatalf("result access: %s/%s", episodeStatus, status)
					}
					if CanDecideLabRequest(tc.actor, request) != tc.ownPatient {
						t.Fatal("approval ownership")
					}
				}
			}
		})
	}
}
