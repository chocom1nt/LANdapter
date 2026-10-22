ALTER TABLE clients
    ADD COLUMN IF NOT EXISTS token_hash TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_clients_token_hash
    ON clients (token_hash)
    WHERE token_hash IS NOT NULL;