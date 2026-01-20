-- +goose Up

-- Step 1: Add address columns and client_id to inspections
ALTER TABLE inspections
    ADD COLUMN address_line1 VARCHAR(255),
    ADD COLUMN address_line2 VARCHAR(255),
    ADD COLUMN city VARCHAR(100),
    ADD COLUMN state VARCHAR(50),
    ADD COLUMN postal_code VARCHAR(20),
    ADD COLUMN client_id UUID REFERENCES clients(id);

-- Step 2: Migrate existing data from linked sites
UPDATE inspections i
SET
    address_line1 = s.address_line1,
    address_line2 = s.address_line2,
    city = s.city,
    state = s.state,
    postal_code = s.postal_code,
    client_id = s.client_id
FROM sites s
WHERE i.site_id = s.id;

-- Step 3: Set default values for inspections without a site
-- (These will need to be updated by users, but allows migration to proceed)
UPDATE inspections
SET
    address_line1 = 'Address not specified',
    city = 'Unknown',
    state = 'Unknown',
    postal_code = '00000'
WHERE address_line1 IS NULL;

-- Step 4: Make required address fields NOT NULL
ALTER TABLE inspections
    ALTER COLUMN address_line1 SET NOT NULL,
    ALTER COLUMN city SET NOT NULL,
    ALTER COLUMN state SET NOT NULL,
    ALTER COLUMN postal_code SET NOT NULL;

-- Step 5: Drop the site_id column
ALTER TABLE inspections DROP COLUMN site_id;

-- Step 6: Add index on client_id for lookups
CREATE INDEX idx_inspections_client_id ON inspections(client_id);

-- Step 7: Drop the sites table
DROP TABLE IF EXISTS sites;

-- +goose Down

-- Recreate sites table
CREATE TABLE sites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(255) NOT NULL,
    address_line1 VARCHAR(255) NOT NULL,
    address_line2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(50) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    client_name VARCHAR(255),
    client_email VARCHAR(255),
    client_phone VARCHAR(50),
    notes TEXT,
    client_id UUID REFERENCES clients(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_sites_user_id ON sites(user_id);
CREATE INDEX idx_sites_client_id ON sites(client_id);

-- Add site_id back to inspections
ALTER TABLE inspections ADD COLUMN site_id UUID REFERENCES sites(id);

-- Note: Data migration back is not possible - site relationships are lost
-- Addresses are now embedded in inspections

-- Drop the inspection address columns and client_id
DROP INDEX IF EXISTS idx_inspections_client_id;
ALTER TABLE inspections
    DROP COLUMN IF EXISTS address_line1,
    DROP COLUMN IF EXISTS address_line2,
    DROP COLUMN IF EXISTS city,
    DROP COLUMN IF EXISTS state,
    DROP COLUMN IF EXISTS postal_code,
    DROP COLUMN IF EXISTS client_id;
