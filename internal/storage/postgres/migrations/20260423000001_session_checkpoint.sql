-- +goose Up
CREATE SCHEMA IF NOT EXISTS flow;

CREATE TABLE IF NOT EXISTS flow.session_checkpoint
(
    id            uuid        NOT NULL DEFAULT gen_random_uuid(),
    connection_id text        NOT NULL,
    domain_id     bigint      NOT NULL,
    channel       smallint    NOT NULL,
    schema_id     int         NOT NULL,
    app_id        text        NOT NULL,
    variables     jsonb,
    status        text        NOT NULL DEFAULT 'active',
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    closed_at     timestamptz,
    CONSTRAINT flow_session_checkpoint_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS flow_session_checkpoint_conn_idx
    ON flow.session_checkpoint (connection_id);

CREATE INDEX IF NOT EXISTS flow_session_checkpoint_app_active_idx
    ON flow.session_checkpoint (app_id, updated_at)
    WHERE status = 'active';

-- +goose Down
DROP TABLE IF EXISTS flow.session_checkpoint;
