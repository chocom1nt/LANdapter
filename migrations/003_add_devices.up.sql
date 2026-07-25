CREATE TABLE client_devices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    device_type TEXT NOT NULL,      -- mouse, keyboard, printer, etc.
    manufacturer TEXT,
    model TEXT,
    info JSONB
);