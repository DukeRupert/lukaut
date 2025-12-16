-- name: CreateViolationRegulation :one
INSERT INTO violation_regulations (
    violation_id,
    regulation_id,
    relevance_score,
    ai_explanation,
    is_primary
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListRegulationsByViolationID :many
SELECT
    r.*,
    vr.relevance_score,
    vr.ai_explanation,
    vr.is_primary
FROM violation_regulations vr
JOIN regulations r ON r.id = vr.regulation_id
WHERE vr.violation_id = $1
ORDER BY vr.is_primary DESC, vr.relevance_score DESC;

-- name: DeleteViolationRegulationsByViolationID :exec
DELETE FROM violation_regulations
WHERE violation_id = $1;

-- name: AddRegulationToViolation :one
INSERT INTO violation_regulations (
    violation_id,
    regulation_id,
    relevance_score,
    ai_explanation,
    is_primary
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (violation_id, regulation_id) DO NOTHING
RETURNING *;

-- name: RemoveRegulationFromViolation :exec
DELETE FROM violation_regulations
WHERE violation_id = $1 AND regulation_id = $2;

-- name: GetViolationRegulation :one
SELECT * FROM violation_regulations
WHERE violation_id = $1 AND regulation_id = $2;
