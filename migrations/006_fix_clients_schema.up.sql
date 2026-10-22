-- Repair client columns for databases created with older migration states.
-- Safe to re-run. Does not delete or reset existing records.
ALTER TABLE clients ADD COLUMN IF NOT EXISTS mac TEXT;

UPDATE clients SET mac = '' WHERE mac IS NULL;
ALTER TABLE clients ALTER COLUMN mac SET DEFAULT '';
ALTER TABLE clients ALTER COLUMN mac SET NOT NULL;

UPDATE clients SET online = FALSE WHERE online IS NULL;
ALTER TABLE clients ALTER COLUMN online SET DEFAULT FALSE;
ALTER TABLE clients ALTER COLUMN online SET NOT NULL;

UPDATE clients SET last_seen = NOW() WHERE last_seen IS NULL;
ALTER TABLE clients ALTER COLUMN last_seen SET DEFAULT NOW();
ALTER TABLE clients ALTER COLUMN last_seen SET NOT NULL;
