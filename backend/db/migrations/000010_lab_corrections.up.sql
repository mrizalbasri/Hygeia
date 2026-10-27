BEGIN;

CREATE TABLE lab_result_corrections (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    lab_result_id uuid NOT NULL REFERENCES lab_results(id) ON DELETE RESTRICT,
    corrected_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reason text NOT NULL CHECK (length(trim(reason)) BETWEEN 5 AND 500),
    value text NOT NULL CHECK (length(trim(value)) BETWEEN 1 AND 500),
    unit text NOT NULL CHECK (length(unit) <= 100),
    reference_range text NOT NULL CHECK (length(reference_range) <= 200),
    flag text NOT NULL CHECK (flag IN ('normal', 'high', 'low', 'abnormal', 'unknown')),
    note text NOT NULL DEFAULT '' CHECK (length(note) <= 2000),
    corrected_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX lab_result_corrections_result_idx ON lab_result_corrections(lab_result_id, corrected_at DESC);

CREATE TRIGGER lab_result_corrections_immutable BEFORE UPDATE OR DELETE ON lab_result_corrections
    FOR EACH ROW EXECUTE FUNCTION reject_lab_result_mutation();
CREATE TRIGGER lab_result_corrections_no_truncate BEFORE TRUNCATE ON lab_result_corrections
    FOR EACH STATEMENT EXECUTE FUNCTION reject_lab_result_mutation();

COMMIT;
