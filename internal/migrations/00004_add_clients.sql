-- +goose Up

-- Clients table (reusable across sites)
CREATE TABLE clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    address_line1 VARCHAR(255),
    address_line2 VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(50),
    postal_code VARCHAR(20),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_clients_user_id ON clients(user_id);
CREATE INDEX idx_clients_name ON clients(user_id, name);

-- Add client_id foreign key to sites
ALTER TABLE sites ADD COLUMN client_id UUID REFERENCES clients(id);
CREATE INDEX idx_sites_client_id ON sites(client_id);

-- +goose StatementBegin
-- Migrate existing client data from sites to clients table
-- Creates a client for each unique (user_id, client_name) combination
DO $$
DECLARE
    site_row RECORD;
    new_client_id UUID;
BEGIN
    -- Create clients from distinct site client info
    FOR site_row IN
        SELECT DISTINCT ON (user_id, client_name)
            user_id,
            client_name,
            client_email,
            client_phone
        FROM sites
        WHERE client_name IS NOT NULL AND client_name != ''
    LOOP
        INSERT INTO clients (user_id, name, email, phone)
        VALUES (site_row.user_id, site_row.client_name, site_row.client_email, site_row.client_phone)
        RETURNING id INTO new_client_id;

        -- Update all sites with this client_name to reference the new client
        UPDATE sites
        SET client_id = new_client_id
        WHERE user_id = site_row.user_id
          AND client_name = site_row.client_name;
    END LOOP;
END $$;
-- +goose StatementEnd

-- Note: Keeping client_name, client_email, client_phone columns for backward compatibility
-- They can be dropped in a future migration after verifying data integrity

-- +goose Down

-- Clear client_id references first
UPDATE sites SET client_id = NULL;

-- Drop the index and column
DROP INDEX IF EXISTS idx_sites_client_id;
ALTER TABLE sites DROP COLUMN IF EXISTS client_id;

-- Drop clients table
DROP INDEX IF EXISTS idx_clients_name;
DROP INDEX IF EXISTS idx_clients_user_id;
DROP TABLE IF EXISTS clients;
