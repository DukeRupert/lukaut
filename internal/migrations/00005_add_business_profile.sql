-- +goose Up
-- +goose StatementBegin

-- Add business profile fields to users table
-- These fields store the inspector's business information for report generation
ALTER TABLE users
ADD COLUMN business_name VARCHAR(255),
ADD COLUMN business_email VARCHAR(255),
ADD COLUMN business_phone VARCHAR(50),
ADD COLUMN business_address_line1 VARCHAR(255),
ADD COLUMN business_address_line2 VARCHAR(255),
ADD COLUMN business_city VARCHAR(100),
ADD COLUMN business_state VARCHAR(50),
ADD COLUMN business_postal_code VARCHAR(20),
ADD COLUMN business_license_number VARCHAR(100),
ADD COLUMN business_logo_url VARCHAR(512);

-- Add comment for documentation
COMMENT ON COLUMN users.business_name IS 'Business/company name for report headers';
COMMENT ON COLUMN users.business_email IS 'Business contact email for reports';
COMMENT ON COLUMN users.business_phone IS 'Business phone number for reports';
COMMENT ON COLUMN users.business_address_line1 IS 'Business street address';
COMMENT ON COLUMN users.business_address_line2 IS 'Business suite/unit number';
COMMENT ON COLUMN users.business_city IS 'Business city';
COMMENT ON COLUMN users.business_state IS 'Business state';
COMMENT ON COLUMN users.business_postal_code IS 'Business ZIP/postal code';
COMMENT ON COLUMN users.business_license_number IS 'Professional license or certification number';
COMMENT ON COLUMN users.business_logo_url IS 'URL to uploaded business logo image';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE users
DROP COLUMN IF EXISTS business_name,
DROP COLUMN IF EXISTS business_email,
DROP COLUMN IF EXISTS business_phone,
DROP COLUMN IF EXISTS business_address_line1,
DROP COLUMN IF EXISTS business_address_line2,
DROP COLUMN IF EXISTS business_city,
DROP COLUMN IF EXISTS business_state,
DROP COLUMN IF EXISTS business_postal_code,
DROP COLUMN IF EXISTS business_license_number,
DROP COLUMN IF EXISTS business_logo_url;

-- +goose StatementEnd
