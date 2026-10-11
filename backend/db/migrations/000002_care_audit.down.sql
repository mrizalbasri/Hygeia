BEGIN;
DROP TABLE audit_events;
DROP FUNCTION reject_audit_mutation();
DROP TABLE audit_chain_heads;
DROP TABLE care_episodes;
COMMIT;
