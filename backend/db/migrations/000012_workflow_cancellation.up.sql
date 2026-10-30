BEGIN;

ALTER TABLE lab_requests
    ADD COLUMN cancellation_note text CHECK (length(trim(cancellation_note)) BETWEEN 5 AND 500),
    DROP CONSTRAINT lab_requests_cancellation_reason_check,
    DROP CONSTRAINT lab_requests_check,
    ADD CONSTRAINT lab_requests_cancellation_reason_check CHECK (cancellation_reason IN ('patient_rejected','episode_ended','patient_cancelled','doctor_cancelled','provider_unavailable')),
    ADD CONSTRAINT lab_requests_check CHECK (
        (status='pending_patient_approval' AND approved_at IS NULL AND completed_at IS NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL AND cancellation_note IS NULL) OR
        (status IN ('sent','accepted','in_progress') AND approved_at IS NOT NULL AND completed_at IS NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL AND cancellation_note IS NULL) OR
        (status='completed' AND approved_at IS NOT NULL AND completed_at IS NOT NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL AND cancellation_note IS NULL) OR
        (status='cancelled' AND cancellation_reason IS NOT NULL AND completed_at IS NULL AND cancelled_at IS NOT NULL AND (
            (cancellation_reason IN ('patient_rejected','episode_ended') AND approved_at IS NULL AND cancellation_note IS NULL) OR
            (cancellation_reason IN ('patient_cancelled','doctor_cancelled','provider_unavailable') AND cancellation_note IS NOT NULL)
        ))
    );

ALTER TABLE prescriptions
    ADD COLUMN cancellation_note text CHECK (length(trim(cancellation_note)) BETWEEN 5 AND 500),
    DROP CONSTRAINT prescriptions_cancellation_reason_check,
    DROP CONSTRAINT prescriptions_check,
    ADD CONSTRAINT prescriptions_cancellation_reason_check CHECK (cancellation_reason IN ('patient_rejected','episode_ended','patient_cancelled','doctor_cancelled','provider_unavailable')),
    ADD CONSTRAINT prescriptions_check CHECK (
        (status='pending_patient_approval' AND pharmacy_org_id IS NULL AND approved_at IS NULL AND completed_at IS NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL AND cancellation_note IS NULL) OR
        (status IN ('sent','accepted','preparing','ready') AND pharmacy_org_id IS NOT NULL AND approved_at IS NOT NULL AND completed_at IS NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL AND cancellation_note IS NULL) OR
        (status='completed' AND pharmacy_org_id IS NOT NULL AND approved_at IS NOT NULL AND completed_at IS NOT NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL AND cancellation_note IS NULL) OR
        (status='cancelled' AND cancellation_reason IS NOT NULL AND completed_at IS NULL AND cancelled_at IS NOT NULL AND (
            (cancellation_reason IN ('patient_rejected','episode_ended') AND pharmacy_org_id IS NULL AND approved_at IS NULL AND cancellation_note IS NULL) OR
            (cancellation_reason IN ('patient_cancelled','doctor_cancelled','provider_unavailable') AND cancellation_note IS NOT NULL AND ((approved_at IS NULL AND pharmacy_org_id IS NULL) OR (approved_at IS NOT NULL AND pharmacy_org_id IS NOT NULL)))
        ))
    );

COMMIT;
