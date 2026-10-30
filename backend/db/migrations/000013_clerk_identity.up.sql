BEGIN;
ALTER TABLE users ADD COLUMN clerk_user_id text UNIQUE CHECK (clerk_user_id ~ '^user_[A-Za-z0-9]+$');
ALTER TABLE sessions ADD COLUMN clerk_session_id text CHECK (clerk_session_id ~ '^sess_[A-Za-z0-9]+$');
CREATE INDEX sessions_clerk_idx ON sessions(clerk_session_id) WHERE clerk_session_id IS NOT NULL;
COMMIT;
