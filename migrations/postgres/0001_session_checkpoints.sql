-- Session checkpoints for stateful channel recovery (chat, im, email).
-- Allows the service to resume or notify clients after a process restart.

CREATE TABLE IF NOT EXISTS flow.session_checkpoint
(
    id            uuid        NOT NULL DEFAULT gen_random_uuid(),
    connection_id text        NOT NULL,
    domain_id     bigint      NOT NULL,
    channel       smallint    NOT NULL, -- maps to model.ConnectionType
    schema_id     int         NOT NULL,
    app_id        text        NOT NULL, -- owning service instance (NodeId)
    variables     jsonb,
    status        text        NOT NULL DEFAULT 'active',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    closed_at     timestamptz,
    CONSTRAINT flow_session_checkpoint_pkey PRIMARY KEY (id)
);

-- fast lookup by connection to close on flow end
CREATE INDEX IF NOT EXISTS flow_session_checkpoint_conn_idx
    ON flow.session_checkpoint (connection_id);

-- recovery worker scans active rows per app ordered by staleness
CREATE INDEX IF NOT EXISTS flow_session_checkpoint_app_active_idx
    ON flow.session_checkpoint (app_id, updated_at)
    WHERE status = 'active';
