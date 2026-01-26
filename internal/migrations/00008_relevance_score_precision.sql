-- +goose Up
-- Widen relevance_score precision to accommodate full-text search ranks.
-- DECIMAL(3,2) only allows 2 decimal places; DECIMAL(7,6) allows up to 6.
ALTER TABLE violation_regulations ALTER COLUMN relevance_score TYPE DECIMAL(7,6);

-- +goose Down
ALTER TABLE violation_regulations ALTER COLUMN relevance_score TYPE DECIMAL(3,2);
