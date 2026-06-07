CREATE TABLE IF NOT EXISTS children (
  id UUID PRIMARY KEY,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  color TEXT NOT NULL DEFAULT '#8fcfb0',
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_children_user_id ON children(user_id);

ALTER TABLE letters
  ADD COLUMN IF NOT EXISTS child_id UUID REFERENCES children(id) ON DELETE SET NULL;

ALTER TABLE manual_events
  ADD COLUMN IF NOT EXISTS child_id UUID REFERENCES children(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_letters_child_id ON letters(child_id);
CREATE INDEX IF NOT EXISTS idx_manual_events_child_id ON manual_events(child_id);
