BEGIN;

ALTER TABLE patients
    DROP CONSTRAINT patients_emergency_contact_pair,
    DROP CONSTRAINT patients_birth_date_year,
    DROP CONSTRAINT patients_birth_date_range,
    DROP COLUMN emergency_contact_phone,
    DROP COLUMN emergency_contact_name,
    DROP COLUMN address,
    DROP COLUMN phone,
    DROP COLUMN birth_date;

COMMIT;
