DROP INDEX IF EXISTS idx_clients_token_hash;

ALTER TABLE clients
    DROP COLUMN IF EXISTS token_hash;