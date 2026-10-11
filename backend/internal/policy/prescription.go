package policy

import "github.com/mrizalbasri/Hygeia/backend/internal/auth"

type Prescription struct {
	Episode       Episode
	PharmacyOrgID string
	Status        string
}

func IsAssignedPharmacy(actor auth.User, prescription Prescription) bool {
	return actor.Role == "pharmacy" && actor.Organization != nil &&
		actor.Organization.Type == "pharmacy" && actor.Organization.ID == prescription.PharmacyOrgID
}

func CanProcessPrescription(actor auth.User, prescription Prescription) bool {
	return IsAssignedPharmacy(actor, prescription) && IsActivePrescription(prescription.Status)
}

func IsActivePrescription(status string) bool {
	switch status {
	case "sent", "accepted", "preparing", "ready":
		return true
	default:
		return false
	}
}

func CanViewPrescription(actor auth.User, prescription Prescription) bool {
	return IsEpisodePatient(actor, prescription.Episode) || HasActiveEpisode(actor, prescription.Episode) || CanProcessPrescription(actor, prescription)
}

func CanDecidePrescription(actor auth.User, prescription Prescription) bool {
	return IsEpisodePatient(actor, prescription.Episode)
}

func CanTransitionPrescription(current, next string) bool {
	return (current == "sent" && next == "accepted") ||
		(current == "accepted" && next == "preparing") ||
		(current == "preparing" && next == "ready") ||
		(current == "ready" && next == "completed")
}
