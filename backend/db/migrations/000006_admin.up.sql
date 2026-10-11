BEGIN;
CREATE TABLE admin_activity (
 id bigserial PRIMARY KEY,
 actor_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 action text NOT NULL CHECK(action IN ('organization.create','organization.rename','user.create','user.update')),
 resource_id uuid NOT NULL,
 occurred_at timestamptz NOT NULL DEFAULT clock_timestamp()
);
CREATE TRIGGER admin_activity_immutable BEFORE UPDATE OR DELETE ON admin_activity
 FOR EACH ROW EXECUTE FUNCTION reject_audit_mutation();
CREATE TRIGGER admin_activity_no_truncate BEFORE TRUNCATE ON admin_activity
 FOR EACH STATEMENT EXECUTE FUNCTION reject_audit_mutation();
COMMIT;
