BEGIN;
DROP TABLE prescription_items;
DROP FUNCTION reject_prescription_item_mutation();
DROP TABLE prescriptions;
COMMIT;
