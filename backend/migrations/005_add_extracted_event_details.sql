ALTER TABLE extracted_events
  ADD COLUMN IF NOT EXISTS belongings TEXT,
  ADD COLUMN IF NOT EXISTS submission_deadline DATE;
