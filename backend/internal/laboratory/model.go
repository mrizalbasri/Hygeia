package laboratory

import (
	"time"

	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/policy"
)

type Request struct {
	ID                 string     `json:"id"`
	EpisodeID          string     `json:"episode_id"`
	PatientRef         string     `json:"patient_ref"`
	LabOrgID           string     `json:"lab_org_id"`
	LabName            string     `json:"lab_name"`
	TestCode           string     `json:"test_code"`
	TestName           string     `json:"test_name"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	ApprovedAt         *time.Time `json:"approved_at"`
	CompletedAt        *time.Time `json:"completed_at"`
	CancelledAt        *time.Time `json:"cancelled_at"`
	CancellationReason *string    `json:"cancellation_reason"`
	ResultID           *string    `json:"result_id"`
	patientID          string
	episode            policy.Episode
}

func (r Request) access() policy.LabRequest {
	return policy.LabRequest{Episode: r.episode, LabOrgID: r.LabOrgID, Status: r.Status}
}

// Separate type prevents future metadata additions from leaking into lab responses.
type QueueRequest struct {
	ID         string `json:"id"`
	PatientRef string `json:"patient_ref"`
	TestCode   string `json:"test_code"`
	TestName   string `json:"test_name"`
	Status     string `json:"status"`
}

func (r Request) view(actor auth.User) any {
	if actor.Role == "lab" {
		return QueueRequest{r.ID, r.PatientRef, r.TestCode, r.TestName, r.Status}
	}
	return r
}

type CreateInput struct {
	EpisodeID string `json:"episode_id"`
	LabOrgID  string `json:"lab_org_id"`
	TestCode  string `json:"test_code"`
	TestName  string `json:"test_name"`
}

type ResultInput struct {
	Value          string `json:"value"`
	Unit           string `json:"unit"`
	ReferenceRange string `json:"reference_range"`
	Flag           string `json:"flag"`
	Note           string `json:"note"`
}

type Result struct {
	ID           string `json:"id"`
	LabRequestID string `json:"lab_request_id"`
	TestCode     string `json:"test_code"`
	TestName     string `json:"test_name"`
	ResultInput
	SubmittedAt time.Time `json:"submitted_at"`
}

type Receipt struct {
	ID           string    `json:"id"`
	LabRequestID string    `json:"lab_request_id"`
	Status       string    `json:"status"`
	SubmittedAt  time.Time `json:"submitted_at"`
}

type Laboratory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
