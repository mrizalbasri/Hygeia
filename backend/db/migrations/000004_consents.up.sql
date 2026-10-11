BEGIN;
CREATE TABLE consents (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id uuid NOT NULL REFERENCES patients(id) ON DELETE RESTRICT,
    grantee_user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    grantee_org_id uuid NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    resource_type text NOT NULL DEFAULT 'lab_result' CHECK (resource_type='lab_result'),
    resource_id uuid NOT NULL REFERENCES lab_results(id) ON DELETE RESTRICT,
    allowed_action text NOT NULL DEFAULT 'view' CHECK (allowed_action='view'),
    purpose text NOT NULL CHECK (length(trim(purpose)) BETWEEN 1 AND 1000),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','active','denied','expired','revoked')),
    requested_at timestamptz NOT NULL DEFAULT now(),
    decided_at timestamptz,
    expires_at timestamptz,
    revoked_at timestamptz,
    CHECK (
        (status='pending' AND decided_at IS NULL AND expires_at IS NULL AND revoked_at IS NULL) OR
        (status='denied' AND decided_at IS NOT NULL AND expires_at IS NULL AND revoked_at IS NULL) OR
        (status IN ('active','expired') AND decided_at IS NOT NULL AND expires_at IS NOT NULL AND expires_at>decided_at AND revoked_at IS NULL) OR
        (status='revoked' AND decided_at IS NOT NULL AND expires_at IS NOT NULL AND expires_at>decided_at AND revoked_at IS NOT NULL)
    )
);
CREATE UNIQUE INDEX consents_open_grant_idx ON consents(grantee_user_id,grantee_org_id,resource_id)
    WHERE status IN ('pending','active');
CREATE INDEX consents_patient_list_idx ON consents(patient_id,requested_at DESC,id DESC);
CREATE INDEX consents_doctor_list_idx ON consents(grantee_user_id,grantee_org_id,requested_at DESC,id DESC);
CREATE INDEX consents_resource_idx ON consents(resource_id);
CREATE INDEX consents_org_idx ON consents(grantee_org_id);
COMMIT;
