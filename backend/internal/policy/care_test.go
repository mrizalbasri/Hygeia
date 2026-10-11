package policy

import (
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"testing"
)

func TestEpisodeAccessMatrix(t *testing.T) {
	episode := Episode{DoctorID: "doctor", PatientUserID: "patient", OrganizationID: "clinic", Status: "active"}
	cases := []struct {
		name                        string
		actor                       auth.User
		view, decide, end, clinical bool
	}{
		{"patient", auth.User{ID: "patient", Role: "patient"}, true, true, true, false},
		{"other patient", auth.User{ID: "other", Role: "patient"}, false, false, false, false},
		{"inviting doctor", auth.User{ID: "doctor", Role: "doctor", Organization: &auth.Organization{ID: "clinic", Type: "clinic"}}, true, false, true, true},
		{"same clinic colleague", auth.User{ID: "colleague", Role: "doctor", Organization: &auth.Organization{ID: "clinic", Type: "clinic"}}, false, false, false, false},
		{"transferred doctor", auth.User{ID: "doctor", Role: "doctor", Organization: &auth.Organization{ID: "other", Type: "clinic"}}, false, false, false, false},
		{"doctor without membership", auth.User{ID: "doctor", Role: "doctor"}, false, false, false, false},
		{"lab", auth.User{ID: "lab", Role: "lab"}, false, false, false, false},
		{"pharmacy", auth.User{ID: "pharmacy", Role: "pharmacy"}, false, false, false, false},
		{"admin", auth.User{ID: "admin", Role: "admin"}, false, false, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if CanViewEpisode(tc.actor, episode) != tc.view || CanTransitionEpisode(tc.actor, episode, "accept") != tc.decide || CanTransitionEpisode(tc.actor, episode, "reject") != tc.decide || CanTransitionEpisode(tc.actor, episode, "end") != tc.end || HasActiveEpisode(tc.actor, episode) != tc.clinical {
				t.Fatal("unexpected access decision")
			}
			for _, status := range []string{"pending", "rejected", "ended"} {
				copy := episode
				copy.Status = status
				if HasActiveEpisode(tc.actor, copy) {
					t.Fatal("inactive episode grants clinical access")
				}
			}
			if CanTransitionEpisode(tc.actor, episode, "other") {
				t.Fatal("unknown action allowed")
			}
		})
	}
}
