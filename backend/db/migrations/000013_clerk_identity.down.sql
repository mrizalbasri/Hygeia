BEGIN;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM users WHERE clerk_user_id IS NOT NULL) THEN
        RAISE EXCEPTION 'Clerk identities exist; refuse a lossy identity downgrade';
    END IF;
END $$;
ALTER TABLE sessions DROP COLUMN clerk_session_id;
ALTER TABLE users DROP COLUMN clerk_user_id;
COMMIT;
