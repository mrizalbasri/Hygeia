BEGIN;

CREATE TABLE prescriptions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_id uuid NOT NULL REFERENCES care_episodes(id) ON DELETE RESTRICT,
    pharmacy_org_id uuid REFERENCES organizations(id) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'pending_patient_approval'
        CHECK (status IN ('pending_patient_approval','sent','accepted','preparing','ready','completed','cancelled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    approved_at timestamptz,
    completed_at timestamptz,
    cancelled_at timestamptz,
    cancellation_reason text CHECK (cancellation_reason IN ('patient_rejected','episode_ended')),
    CHECK (
        (status='pending_patient_approval' AND pharmacy_org_id IS NULL AND approved_at IS NULL AND completed_at IS NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL) OR
        (status IN ('sent','accepted','preparing','ready') AND pharmacy_org_id IS NOT NULL AND approved_at IS NOT NULL AND completed_at IS NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL) OR
        (status='completed' AND pharmacy_org_id IS NOT NULL AND approved_at IS NOT NULL AND completed_at IS NOT NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL) OR
        (status='cancelled' AND pharmacy_org_id IS NULL AND approved_at IS NULL AND completed_at IS NULL AND cancelled_at IS NOT NULL AND cancellation_reason IS NOT NULL)
    )
);
CREATE INDEX prescriptions_episode_list_idx ON prescriptions(episode_id,created_at DESC,id DESC);
CREATE INDEX prescriptions_pharmacy_queue_idx ON prescriptions(pharmacy_org_id,created_at DESC,id DESC)
    WHERE status IN ('sent','accepted','preparing','ready');
CREATE INDEX prescriptions_pending_episode_idx ON prescriptions(episode_id,id)
    WHERE status='pending_patient_approval';

CREATE TABLE prescription_items (
    prescription_id uuid NOT NULL REFERENCES prescriptions(id) ON DELETE RESTRICT,
    position smallint NOT NULL CHECK (position BETWEEN 1 AND 20),
    medication_name text NOT NULL CHECK (length(trim(medication_name)) BETWEEN 1 AND 200),
    dosage text NOT NULL CHECK (length(trim(dosage)) BETWEEN 1 AND 200),
    instructions text NOT NULL CHECK (length(trim(instructions)) BETWEEN 1 AND 1000),
    quantity integer NOT NULL CHECK (quantity BETWEEN 1 AND 10000),
    PRIMARY KEY (prescription_id,position)
);

-- Issued medication details cannot be silently rewritten after patient approval.
CREATE FUNCTION reject_prescription_item_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'issued prescription items are immutable';
END;
$$;
CREATE TRIGGER prescription_items_immutable BEFORE UPDATE OR DELETE ON prescription_items
    FOR EACH ROW EXECUTE FUNCTION reject_prescription_item_mutation();
CREATE TRIGGER prescription_items_no_truncate BEFORE TRUNCATE ON prescription_items
    FOR EACH STATEMENT EXECUTE FUNCTION reject_prescription_item_mutation();

COMMIT;
