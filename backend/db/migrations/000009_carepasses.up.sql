BEGIN;
CREATE TABLE carepasses (
 id uuid PRIMARY KEY DEFAULT gen_random_uuid(), patient_id uuid NOT NULL REFERENCES patients(id),
 resource_type text NOT NULL CHECK(resource_type IN ('lab_result','prescription')), resource_id uuid NOT NULL,
 token_hash bytea NOT NULL UNIQUE CHECK(octet_length(token_hash)=32),
 created_at timestamptz NOT NULL, expires_at timestamptz NOT NULL,
 revoked_at timestamptz, grantee_user_id uuid REFERENCES users(id), grantee_org_id uuid REFERENCES organizations(id), redeemed_at timestamptz,
 CHECK(expires_at=created_at+interval '15 minutes' OR expires_at=created_at+interval '60 minutes'),
 CHECK((grantee_user_id IS NULL AND grantee_org_id IS NULL AND redeemed_at IS NULL) OR
 (grantee_user_id IS NOT NULL AND grantee_org_id IS NOT NULL AND redeemed_at IS NOT NULL))
);
CREATE INDEX carepasses_owner_idx ON carepasses(patient_id,created_at DESC,id);
CREATE INDEX carepasses_grant_idx ON carepasses(grantee_user_id,grantee_org_id,resource_type,resource_id,expires_at) WHERE revoked_at IS NULL;
COMMIT;
