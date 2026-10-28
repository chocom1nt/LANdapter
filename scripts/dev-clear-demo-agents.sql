-- DEVELOPMENT ONLY: remove only the deterministic demo clients.
DELETE FROM clients WHERE id::text LIKE 'd0000000-0000-4000-8000-00000000000%';
