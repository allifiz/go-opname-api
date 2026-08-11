-- name: ListRoles :many
SELECT id, code, name, created_at
FROM roles
ORDER BY name ASC;

-- name: GetRoleByCode :one
SELECT id, code, name, created_at
FROM roles
WHERE code = $1;

-- name: ListUnits :many
SELECT id, code, name, created_at
FROM units
ORDER BY code ASC;

-- name: GetUnitByID :one
SELECT id, code, name, created_at
FROM units
WHERE id = $1;

-- name: CreateMaterial :one
INSERT INTO materials (
    name,
    unit_id,
    created_by,
    updated_by
) VALUES ($1, $2, $3, $3)
RETURNING *;

-- name: GetMaterialByID :one
SELECT
    m.id,
    m.name,
    m.unit_id,
    u.code AS unit_code,
    u.name AS unit_name,
    m.is_active,
    m.created_by,
    m.updated_by,
    m.created_at,
    m.updated_at
FROM materials m
JOIN units u ON u.id = m.unit_id
WHERE m.id = $1;

-- name: ListMaterials :many
SELECT
    m.id,
    m.name,
    m.unit_id,
    u.code AS unit_code,
    u.name AS unit_name,
    m.is_active,
    m.created_by,
    m.updated_by,
    m.created_at,
    m.updated_at
FROM materials m
JOIN units u ON u.id = m.unit_id
ORDER BY m.name ASC;

-- name: UpdateMaterial :one
UPDATE materials
SET
    name = $2,
    unit_id = $3,
    is_active = $4,
    updated_by = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;
