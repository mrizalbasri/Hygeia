BEGIN;

ALTER TABLE patients
    ADD COLUMN birth_date date,
    ADD COLUMN phone text NOT NULL DEFAULT '' CHECK (length(phone) <= 32),
    ADD COLUMN address text NOT NULL DEFAULT '' CHECK (length(address) <= 1000),
    ADD COLUMN emergency_contact_name text NOT NULL DEFAULT '' CHECK (length(emergency_contact_name) <= 200),
    ADD COLUMN emergency_contact_phone text NOT NULL DEFAULT '' CHECK (length(emergency_contact_phone) <= 32),
    ADD CONSTRAINT patients_birth_date_range CHECK (birth_date >= DATE '1900-01-01' AND birth_date <= DATE '2100-12-31'),
    ADD CONSTRAINT patients_birth_date_year CHECK (birth_date IS NULL OR (birth_year IS NOT NULL AND birth_year = EXTRACT(YEAR FROM birth_date))),
    ADD CONSTRAINT patients_emergency_contact_pair CHECK ((emergency_contact_name = '') = (emergency_contact_phone = ''));

COMMIT;
