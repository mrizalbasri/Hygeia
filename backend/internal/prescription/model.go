package prescription

import (
	"time"

	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/policy"
)

type Item struct {
	MedicationName string `json:"medication_name"`
	Dosage         string `json:"dosage"`
	Instructions   string `json:"instructions"`
	Quantity       int    `json:"quantity"`
}

type CreateInput struct {
	EpisodeID string `json:"episode_id"`
	Items     []Item `json:"items"`
}

type ApprovalInput struct {
	Decision      string `json:"decision"`
	PharmacyOrgID string `json:"pharmacy_org_id"`
}

type Prescription struct {
	ID                 string     `json:"id"`
	EpisodeID          string     `json:"episode_id"`
	PatientRef         string     `json:"patient_ref"`
	PharmacyOrgID      *string    `json:"pharmacy_org_id"`
	PharmacyName       *string    `json:"pharmacy_name"`
	Items              []Item     `json:"items"`
	Status             string     `json:"status"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	ApprovedAt         *time.Time `json:"approved_at"`
	CompletedAt        *time.Time `json:"completed_at"`
	CancelledAt        *time.Time `json:"cancelled_at"`
	CancellationReason *string    `json:"cancellation_reason"`
	CancellationNote   *string    `json:"cancellation_note"`
	patientID          string
	episode            policy.Episode
}

func (p Prescription) access() policy.Prescription {
	org := ""
	if p.PharmacyOrgID != nil {
		org = *p.PharmacyOrgID
	}
	return policy.Prescription{Episode: p.episode, PharmacyOrgID: org, Status: p.Status}
}

// The pharmacy receives only the data required to fulfil this prescription.
type QueuePrescription struct {
	ID         string `json:"id"`
	PatientRef string `json:"patient_ref"`
	Items      []Item `json:"items"`
	Status     string `json:"status"`
}

func (p Prescription) view(actor auth.User) any {
	if actor.Role == "pharmacy" {
		return QueuePrescription{p.ID, p.PatientRef, p.Items, p.Status}
	}
	return p
}

type Pharmacy struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
