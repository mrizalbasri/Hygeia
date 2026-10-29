BEGIN;
-- Refuse a lossy downgrade when cancellations using the new history exist.
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM lab_requests WHERE cancellation_note IS NOT NULL)
       OR EXISTS (SELECT 1 FROM prescriptions WHERE cancellation_note IS NOT NULL) THEN
        RAISE EXCEPTION 'workflow cancellation history exists; restore a compatible backup before downgrade';
    END IF;
END $$;
ALTER TABLE lab_requests DROP CONSTRAINT lab_requests_check,
    DROP CONSTRAINT lab_requests_cancellation_reason_check,
    DROP COLUMN cancellation_note,
    ADD CONSTRAINT lab_requests_cancellation_reason_check CHECK (cancellation_reason IN ('patient_rejected','episode_ended')),
    ADD CONSTRAINT lab_requests_check CHECK (
        (status='pending_patient_approval' AND approved_at IS NULL AND completed_at IS NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL) OR
        (status IN ('sent','accepted','in_progress') AND approved_at IS NOT NULL AND completed_at IS NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL) OR
        (status='completed' AND approved_at IS NOT NULL AND completed_at IS NOT NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL) OR
        (status='cancelled' AND approved_at IS NULL AND completed_at IS NULL AND cancelled_at IS NOT NULL AND cancellation_reason IS NOT NULL)
    );
ALTER TABLE prescriptions DROP CONSTRAINT prescriptions_check,
    DROP CONSTRAINT prescriptions_cancellation_reason_check,
    DROP COLUMN cancellation_note,
    ADD CONSTRAINT prescriptions_cancellation_reason_check CHECK (cancellation_reason IN ('patient_rejected','episode_ended')),
    ADD CONSTRAINT prescriptions_check CHECK (
        (status='pending_patient_approval' AND pharmacy_org_id IS NULL AND approved_at IS NULL AND completed_at IS NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL) OR
        (status IN ('sent','accepted','preparing','ready') AND pharmacy_org_id IS NOT NULL AND approved_at IS NOT NULL AND completed_at IS NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL) OR
        (status='completed' AND pharmacy_org_id IS NOT NULL AND approved_at IS NOT NULL AND completed_at IS NOT NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL) OR
        (status='cancelled' AND pharmacy_org_id IS NULL AND approved_at IS NULL AND completed_at IS NULL AND cancelled_at IS NOT NULL AND cancellation_reason IS NOT NULL)
    );
COMMIT;
