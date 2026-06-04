CREATE TABLE IF NOT EXISTS manual_events (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  event_date DATE NOT NULL,
  start_at TIMESTAMPTZ,
  end_at TIMESTAMPTZ,
  is_all_day BOOLEAN NOT NULL DEFAULT true,
  location TEXT,
  description TEXT,
  time_zone TEXT NOT NULL DEFAULT 'Asia/Tokyo',
  google_calendar_event_id TEXT,
  status TEXT NOT NULL DEFAULT 'registered',
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_manual_events_user_id ON manual_events(user_id);
CREATE INDEX IF NOT EXISTS idx_manual_events_status ON manual_events(status);
CREATE INDEX IF NOT EXISTS idx_manual_events_google_calendar_event_id ON manual_events(google_calendar_event_id);
