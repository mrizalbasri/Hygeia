BEGIN;

CREATE TABLE lab_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    episode_id uuid NOT NULL REFERENCES care_episodes(id) ON DELETE RESTRICT,
    lab_org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    test_code text NOT NULL CHECK (test_code ~ '^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$'),
    test_name text NOT NULL CHECK (length(trim(test_name)) BETWEEN 1 AND 200),
    status text NOT NULL DEFAULT 'pending_patient_approval'
        CHECK (status IN ('pending_patient_approval','sent','accepted','in_progress','completed','cancelled')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    approved_at timestamptz,
    completed_at timestamptz,
    cancelled_at timestamptz,
    cancellation_reason text CHECK (cancellation_reason IN ('patient_rejected','episode_ended')),
    CHECK (
        (status='pending_patient_approval' AND approved_at IS NULL AND completed_at IS NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL) OR
        (status IN ('sent','accepted','in_progress') AND approved_at IS NOT NULL AND completed_at IS NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL) OR
        (status='completed' AND approved_at IS NOT NULL AND completed_at IS NOT NULL AND cancelled_at IS NULL AND cancellation_reason IS NULL) OR
        (status='cancelled' AND approved_at IS NULL AND completed_at IS NULL AND cancelled_at IS NOT NULL AND cancellation_reason IS NOT NULL)
    )
);
CREATE INDEX lab_requests_episode_list_idx ON lab_requests(episode_id,created_at DESC,id DESC);
CREATE INDEX lab_requests_lab_queue_idx ON lab_requests(lab_org_id,created_at DESC,id DESC)
    WHERE status IN ('sent','accepted','in_progress');
CREATE INDEX lab_requests_pending_episode_idx ON lab_requests(episode_id,id)
    WHERE status='pending_patient_approval';

CREATE TABLE lab_results (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    lab_request_id uuid NOT NULL UNIQUE REFERENCES lab_requests(id) ON DELETE RESTRICT,
    value text NOT NULL CHECK (length(trim(value)) BETWEEN 1 AND 500),
    unit text NOT NULL CHECK (length(unit)<=100),
    reference_range text NOT NULL CHECK (length(reference_range)<=200),
    flag text NOT NULL CHECK (flag IN ('normal','high','low','abnormal','unknown')),
    note text NOT NULL DEFAULT '' CHECK (length(note)<=2000),
    submitted_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    submitted_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX lab_results_submitter_idx ON lab_results(submitted_by);

CREATE FUNCTION reject_lab_result_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'submitted lab results are immutable';
END;
$$;
CREATE TRIGGER lab_results_immutable BEFORE UPDATE OR DELETE ON lab_results
    FOR EACH ROW EXECUTE FUNCTION reject_lab_result_mutation();
CREATE TRIGGER lab_results_no_truncate BEFORE TRUNCATE ON lab_results
    FOR EACH STATEMENT EXECUTE FUNCTION reject_lab_result_mutation();

COMMIT;
