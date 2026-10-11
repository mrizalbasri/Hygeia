package policy

import "github.com/mrizalbasri/Hygeia/backend/internal/auth"

type Episode struct {
	DoctorID       string
	PatientUserID  string
	OrganizationID string
	Status         string
}

func IsEpisodeDoctor(actor auth.User, episode Episode) bool {
	return actor.Role == "doctor" && actor.ID == episode.DoctorID &&
		actor.Organization != nil && actor.Organization.Type == "clinic" &&
		actor.Organization.ID == episode.OrganizationID
}

func IsEpisodePatient(actor auth.User, episode Episode) bool {
	return actor.Role == "patient" && actor.ID == episode.PatientUserID
}

// CanViewEpisode permits relationship metadata, including invitations and history.
// Clinical resources must separately require an active episode (or future consent).
func CanViewEpisode(actor auth.User, episode Episode) bool {
	return IsEpisodeDoctor(actor, episode) || IsEpisodePatient(actor, episode)
}

func CanTransitionEpisode(actor auth.User, episode Episode, action string) bool {
	switch action {
	case "accept", "reject":
		return IsEpisodePatient(actor, episode)
	case "end":
		return CanViewEpisode(actor, episode)
	default:
		return false
	}
}

func HasActiveEpisode(actor auth.User, episode Episode) bool {
	return episode.Status == "active" && IsEpisodeDoctor(actor, episode)
}
