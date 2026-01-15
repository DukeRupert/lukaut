-- +goose Up
-- +goose StatementBegin
-- Update inspections.site_id FK to set null on site deletion
-- This allows sites to be deleted without affecting linked inspections
ALTER TABLE inspections
DROP CONSTRAINT IF EXISTS inspections_site_id_fkey;

ALTER TABLE inspections
ADD CONSTRAINT inspections_site_id_fkey
    FOREIGN KEY (site_id)
    REFERENCES sites(id)
    ON DELETE SET NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Revert to original FK constraint without ON DELETE SET NULL
ALTER TABLE inspections
DROP CONSTRAINT IF EXISTS inspections_site_id_fkey;

ALTER TABLE inspections
ADD CONSTRAINT inspections_site_id_fkey
    FOREIGN KEY (site_id)
    REFERENCES sites(id);
-- +goose StatementEnd
