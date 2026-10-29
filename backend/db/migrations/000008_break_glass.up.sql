BEGIN;
CREATE TABLE break_glass_sessions (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
 patient_id uuid NOT NULL REFERENCES patients(id),
 doctor_user_id uuid NOT NULL REFERENCES users(id),
 organization_id uuid NOT NULL REFERENCES organizations(id),
 reason_code text NOT NULL CHECK(reason_code IN ('unconscious','unreachable_patient','life_threatening')),
 reason_text text NOT NULL CHECK(length(trim(reason_text)) BETWEEN 10 AND 1000),
 starts_at timestamptz NOT NULL,
 expires_at timestamptz NOT NULL CHECK(expires_at=starts_at+interval '60 minutes'),
 reviewed_by uuid REFERENCES users(id), reviewed_at timestamptz,
 review_decision text CHECK(review_decision IN ('appropriate','needs_followup')),
 review_note text,
 CHECK((reviewed_by IS NULL AND reviewed_at IS NULL AND review_decision IS NULL AND review_note IS NULL) OR
 (reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND review_decision IS NOT NULL AND length(trim(review_note)) BETWEEN 10 AND 1000))
);
CREATE INDEX break_glass_doctor_idx ON break_glass_sessions(doctor_user_id,patient_id,expires_at);
CREATE INDEX break_glass_patient_idx ON break_glass_sessions(patient_id,starts_at DESC);
COMMIT;
