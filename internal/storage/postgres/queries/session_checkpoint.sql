-- name: InsertCheckpoint :one
INSERT INTO flow.session_checkpoint
    (connection_id, domain_id, channel, schema_id, app_id, variables, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;

-- name: UpdateCheckpointVars :exec
UPDATE flow.session_checkpoint
   SET variables = $2, updated_at = $3
 WHERE id = $1;

-- name: CloseCheckpoint :exec
UPDATE flow.session_checkpoint
   SET status = 'closed', closed_at = now(), updated_at = now()
 WHERE connection_id = $1 AND status = 'active';

-- name: ListActiveByApp :many
SELECT id, connection_id, domain_id, channel, schema_id, app_id,
       variables, status, created_at, updated_at, closed_at
  FROM flow.session_checkpoint
 WHERE app_id = $1 AND status = 'active';

-- name: ClaimOrphaned :many
UPDATE flow.session_checkpoint
   SET app_id = $1, updated_at = now()
 WHERE status = 'active' AND updated_at < $2
RETURNING id, connection_id, domain_id, channel, schema_id, app_id,
          variables, status, created_at, updated_at, closed_at;

-- name: TouchByApp :exec
UPDATE flow.session_checkpoint
   SET updated_at = now()
 WHERE app_id = $1 AND status = 'active';
