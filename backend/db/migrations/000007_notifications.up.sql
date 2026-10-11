BEGIN;
CREATE TABLE notifications (
 id bigserial PRIMARY KEY,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 source_chain uuid NOT NULL REFERENCES patients(id) ON DELETE RESTRICT,
 source_seq bigint NOT NULL CHECK(source_seq>0),
 action text NOT NULL,
 resource_type text NOT NULL,
 resource_id uuid NOT NULL,
 message text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 read_at timestamptz,
 UNIQUE(user_id,source_chain,source_seq)
);
CREATE INDEX notifications_user_page_idx ON notifications(user_id,id DESC);
CREATE INDEX notifications_unread_idx ON notifications(user_id,id DESC) WHERE read_at IS NULL;
COMMIT;
