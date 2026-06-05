CREATE TABLE IF NOT EXISTS extracted_events (
  id UUID PRIMARY KEY,
  letter_id UUID NOT NULL REFERENCES letters(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  event_date DATE NOT NULL,
  start_time TIME,
  end_time TIME,
  is_all_day BOOLEAN NOT NULL DEFAULT true,
  location TEXT,
  description TEXT,
  confidence NUMERIC(3,2),
  source_text TEXT,
  google_calendar_event_id TEXT,
  status TEXT NOT NULL DEFAULT 'draft',
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_extracted_events_letter_id ON extracted_events(letter_id);
CREATE INDEX IF NOT EXISTS idx_extracted_events_status ON extracted_events(status);
CREATE INDEX IF NOT EXISTS idx_extracted_events_event_date ON extracted_events(event_date);
CREATE INDEX IF NOT EXISTS idx_extracted_events_google_calendar_event_id ON extracted_events(google_calendar_event_id);
