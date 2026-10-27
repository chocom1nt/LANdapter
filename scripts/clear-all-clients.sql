-- DESTRUCTIVE: deletes every client record and rows linked to clients via ON DELETE CASCADE.
-- The clients table includes real agents and demo fixtures. Run only on the intended database.
BEGIN;
DELETE FROM clients;
COMMIT;
