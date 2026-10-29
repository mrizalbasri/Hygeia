// Package patientprofile owns the patient's self-service demographic profile.
// It does not expose profiles to providers or accept a client-supplied identity.
package patientprofile

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/apiutil"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

type Profile struct {
	PatientRef            string  `json:"patient_ref"`
	Email                 string  `json:"email"`
	FullName              string  `json:"full_name"`
	BirthDate             *string `json:"birth_date"`
	BirthYear             *int    `json:"birth_year"`
	Sex                   string  `json:"sex"`
	Phone                 string  `json:"phone"`
	Address               string  `json:"address"`
	EmergencyContactName  string  `json:"emergency_contact_name"`
	EmergencyContactPhone string  `json:"emergency_contact_phone"`
}

// Input is a complete editable snapshot. Empty optional strings clear a value.
// Clearing a full birth date retains the last known year for legacy consumers.
type Input struct {
	FullName              string `json:"full_name"`
	BirthDate             string `json:"birth_date"`
	Sex                   string `json:"sex"`
	Phone                 string `json:"phone"`
	Address               string `json:"address"`
	EmergencyContactName  string `json:"emergency_contact_name"`
	EmergencyContactPhone string `json:"emergency_contact_phone"`
}

func validPhone(value string) bool {
	if value == "" {
		return true
	}
	if len(value) > 32 {
		return false
	}
	digits := 0
	for i, char := range value {
		switch {
		case char >= '0' && char <= '9':
			digits++
		case char == '+' && i == 0:
		case strings.ContainsRune(" ()-.", char):
		default:
			return false
		}
	}
	return digits >= 7 && digits <= 15
}

func (in *Input) normalize() {
	in.FullName = strings.TrimSpace(in.FullName)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Address = strings.TrimSpace(in.Address)
	in.EmergencyContactName = strings.TrimSpace(in.EmergencyContactName)
	in.EmergencyContactPhone = strings.TrimSpace(in.EmergencyContactPhone)
}

func (in Input) valid(now time.Time) bool {
	if !apiutil.Text(in.FullName, 1, 200) || !apiutil.Text(in.Address, 0, 1000) ||
		!apiutil.Text(in.EmergencyContactName, 0, 200) ||
		!validPhone(in.Phone) || !validPhone(in.EmergencyContactPhone) {
		return false
	}
	if (in.EmergencyContactName == "") != (in.EmergencyContactPhone == "") {
		return false
	}
	if in.Sex != "female" && in.Sex != "male" && in.Sex != "other" && in.Sex != "unknown" {
		return false
	}
	if in.BirthDate != "" {
		date, err := time.Parse(time.DateOnly, in.BirthDate)
		if err != nil || date.Year() < 1900 || in.BirthDate > now.UTC().Format(time.DateOnly) {
			return false
		}
	}
	return true
}

type Service struct {
	DB    apiutil.DB
	Audit *audit.Writer
}

func read(ctx context.Context, tx pgx.Tx, userID string) (Profile, error) {
	var p Profile
	err := tx.QueryRow(ctx, `SELECT p.patient_ref,u.email,u.full_name,to_char(p.birth_date,'YYYY-MM-DD'),
		p.birth_year,COALESCE(p.sex,'unknown'),p.phone,p.address,p.emergency_contact_name,p.emergency_contact_phone
		FROM patients p JOIN users u ON u.id=p.user_id WHERE p.user_id=$1`, userID).Scan(
		&p.PatientRef, &p.Email, &p.FullName, &p.BirthDate, &p.BirthYear, &p.Sex,
		&p.Phone, &p.Address, &p.EmergencyContactName, &p.EmergencyContactPhone,
	)
	return p, err
}

func (s *Service) Get(ctx context.Context, token []byte) (Profile, error) {
	var result Profile
	err := apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if actor.Role != "patient" {
			return apiutil.ErrForbidden
		}
		var err error
		result, err = read(ctx, tx, actor.ID)
		return err
	})
	return result, err
}

func (s *Service) Update(ctx context.Context, token []byte, in Input) (Profile, error) {
	in.normalize()
	var result Profile
	err := apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if actor.Role != "patient" {
			return apiutil.ErrForbidden
		}
		if !in.valid(time.Now()) {
			return apiutil.ErrInvalid
		}
		// Serialize with account edits/disables, then recheck the session after
		// waiting. Both the name and patient fields must commit with the audit.
		if _, err := tx.Exec(ctx, `SELECT id FROM users WHERE id=$1 FOR UPDATE`, actor.ID); err != nil {
			return err
		}
		current, err := (auth.Store{DB: tx}).UserBySession(ctx, token)
		if errors.Is(err, pgx.ErrNoRows) {
			return apiutil.ErrUnauthenticated
		}
		if err != nil {
			return err
		}
		if current.Role != "patient" {
			return apiutil.ErrForbidden
		}
		var patientID string
		err = tx.QueryRow(ctx, `UPDATE patients SET
			birth_date=NULLIF($2,'')::date,
			birth_year=CASE WHEN $2='' THEN birth_year ELSE EXTRACT(YEAR FROM $2::date)::smallint END,
			sex=$3,phone=$4,address=$5,emergency_contact_name=$6,emergency_contact_phone=$7
			WHERE user_id=$1 RETURNING id::text`,
			actor.ID, in.BirthDate, in.Sex, in.Phone, in.Address, in.EmergencyContactName, in.EmergencyContactPhone,
		).Scan(&patientID)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE users SET full_name=$2 WHERE id=$1`, actor.ID, in.FullName); err != nil {
			return err
		}
		if err := s.Audit.Append(ctx, tx, current, audit.Event{
			ChainID: patientID, Action: "patient.profile_update", ResourceType: "patient",
			ResourceID: patientID, Outcome: "allow", Basis: "patient_self",
		}); err != nil {
			return err
		}
		result, err = read(ctx, tx, actor.ID)
		return err
	})
	return result, err
}

type Handler struct{ Service *Service }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/me/profile", h.Get)
	mux.HandleFunc("PATCH /api/v1/me/profile", h.Update)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	if _, err := apiutil.Query(r); err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	result, err := h.Service.Get(r.Context(), auth.SessionHash(r))
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{"profile": result})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	if _, err := apiutil.Query(r); err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	var in Input
	if !apiutil.Decode(w, r, &in) {
		return
	}
	result, err := h.Service.Update(r.Context(), auth.SessionHash(r), in)
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	apiutil.JSON(w, http.StatusOK, map[string]any{"profile": result})
}
