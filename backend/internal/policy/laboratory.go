package policy

import "github.com/mrizalbasri/Hygeia/backend/internal/auth"

type LabRequest struct {
	Episode  Episode
	LabOrgID string
	Status   string
}

func IsAssignedLab(actor auth.User, request LabRequest) bool {
	return actor.Role == "lab" && actor.Organization != nil &&
		actor.Organization.Type == "laboratory" && actor.Organization.ID == request.LabOrgID
}

func CanProcessLabRequest(actor auth.User, request LabRequest) bool {
	return IsAssignedLab(actor, request) &&
		(request.Status == "sent" || request.Status == "accepted" || request.Status == "in_progress")
}

func CanViewLabRequest(actor auth.User, request LabRequest) bool {
	return IsEpisodePatient(actor, request.Episode) || HasActiveEpisode(actor, request.Episode) || CanProcessLabRequest(actor, request)
}

func CanDecideLabRequest(actor auth.User, request LabRequest) bool {
	return IsEpisodePatient(actor, request.Episode)
}

func CanViewLabResult(actor auth.User, request LabRequest) bool {
	return request.Status == "completed" &&
		(IsEpisodePatient(actor, request.Episode) || HasActiveEpisode(actor, request.Episode))
}
