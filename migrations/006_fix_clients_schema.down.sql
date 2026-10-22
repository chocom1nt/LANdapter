-- Restore nullable column constraints. Existing data is left intact.
ALTER TABLE clients ALTER COLUMN mac DROP NOT NULL;
ALTER TABLE clients ALTER COLUMN mac DROP DEFAULT;
ALTER TABLE clients ALTER COLUMN online DROP NOT NULL;
ALTER TABLE clients ALTER COLUMN last_seen DROP NOT NULL;
