BEGIN;

CREATE TABLE care_episodes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id uuid NOT NULL REFERENCES patients(id) ON DELETE RESTRICT,
    doctor_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','active','rejected','ended')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    accepted_at timestamptz,
    closed_at timestamptz,
    CHECK (
        (status='pending' AND accepted_at IS NULL AND closed_at IS NULL) OR
        (status='active' AND accepted_at IS NOT NULL AND closed_at IS NULL) OR
        (status='rejected' AND accepted_at IS NULL AND closed_at IS NOT NULL) OR
        (status='ended' AND accepted_at IS NOT NULL AND closed_at IS NOT NULL)
    )
);
CREATE UNIQUE INDEX care_episodes_open_pair_idx ON care_episodes(doctor_id,patient_id)
    WHERE status IN ('pending','active');
CREATE INDEX care_episodes_doctor_list_idx ON care_episodes(doctor_id,created_at DESC,id DESC);
CREATE INDEX care_episodes_patient_list_idx ON care_episodes(patient_id,created_at DESC,id DESC);
CREATE INDEX care_episodes_org_idx ON care_episodes(organization_id);

CREATE TABLE audit_chain_heads (
    chain_id uuid PRIMARY KEY REFERENCES patients(id) ON DELETE RESTRICT,
    last_seq bigint NOT NULL DEFAULT 0 CHECK (last_seq >= 0),
    last_hash bytea NOT NULL CHECK (octet_length(last_hash)=32)
);
CREATE TABLE audit_events (
    chain_id uuid NOT NULL REFERENCES audit_chain_heads(chain_id) ON DELETE RESTRICT,
    seq bigint NOT NULL CHECK (seq > 0),
    occurred_at timestamptz NOT NULL,
    actor_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action text NOT NULL,
    resource_type text NOT NULL,
    resource_id uuid NOT NULL,
    outcome text NOT NULL CHECK (outcome IN ('allow','deny')),
    basis text NOT NULL,
    context jsonb NOT NULL CHECK (jsonb_typeof(context)='object'),
    prev_hash bytea NOT NULL CHECK (octet_length(prev_hash)=32),
    hash bytea NOT NULL CHECK (octet_length(hash)=32),
    PRIMARY KEY(chain_id,seq)
);
CREATE INDEX audit_events_actor_idx ON audit_events(actor_id);

CREATE FUNCTION reject_audit_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit events are append-only';
END;
$$;
CREATE TRIGGER audit_events_immutable BEFORE UPDATE OR DELETE ON audit_events
    FOR EACH ROW EXECUTE FUNCTION reject_audit_mutation();
CREATE TRIGGER audit_events_no_truncate BEFORE TRUNCATE ON audit_events
    FOR EACH STATEMENT EXECUTE FUNCTION reject_audit_mutation();

COMMIT;
