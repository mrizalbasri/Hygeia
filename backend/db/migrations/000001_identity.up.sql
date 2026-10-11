BEGIN;

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL UNIQUE CHECK (email = lower(email) AND length(email) BETWEEN 3 AND 254),
    password_hash text NOT NULL,
    full_name text NOT NULL CHECK (length(trim(full_name)) BETWEEN 1 AND 200),
    role text NOT NULL CHECK (role IN ('patient', 'doctor', 'lab', 'pharmacy', 'admin')),
    created_at timestamptz NOT NULL DEFAULT now(),
    disabled_at timestamptz
);

CREATE TABLE organizations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    type text NOT NULL CHECK (type IN ('clinic', 'laboratory', 'pharmacy')),
    name text NOT NULL CHECK (length(trim(name)) BETWEEN 1 AND 200),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (type, name)
);

CREATE TABLE organization_memberships (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE RESTRICT,
    organization_id uuid NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    staff_role text NOT NULL CHECK (staff_role IN ('doctor', 'lab', 'pharmacy')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX organization_memberships_org_idx ON organization_memberships(organization_id);

CREATE TABLE patients (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES users(id) ON DELETE RESTRICT,
    patient_ref text NOT NULL UNIQUE,
    sex text CHECK (sex IN ('female', 'male', 'other', 'unknown')),
    birth_year smallint CHECK (birth_year BETWEEN 1900 AND 2100),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash bytea PRIMARY KEY CHECK (octet_length(token_hash) = 32),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL CHECK (expires_at > created_at)
);
CREATE INDEX sessions_user_idx ON sessions(user_id);
CREATE INDEX sessions_expiry_idx ON sessions(expires_at);

COMMIT;
