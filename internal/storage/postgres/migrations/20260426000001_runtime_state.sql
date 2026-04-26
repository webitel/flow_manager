-- +goose Up
CREATE TABLE IF NOT EXISTS flow.runtime_state
(
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_id  text        NOT NULL,
    domain_id      bigint      NOT NULL,
    channel        smallint    NOT NULL,
    schema_id      int         NOT NULL,
    schema_version bigint      NOT NULL,
    app_id         text        NOT NULL,

    state          jsonb       NOT NULL,

    status         text        NOT NULL DEFAULT 'running',
    resume_key     text,
    fail_reason    text,

    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    suspended_at   timestamptz,
    completed_at   timestamptz
);

CREATE INDEX IF NOT EXISTS runtime_state_resume_key_idx
    ON flow.runtime_state (resume_key)
    WHERE resume_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS runtime_state_app_active_idx
    ON flow.runtime_state (app_id, updated_at)
    WHERE status = 'running' OR status = 'suspended';

CREATE INDEX IF NOT EXISTS runtime_state_conn_idx
    ON flow.runtime_state (connection_id);

-- +goose Down
DROP TABLE IF EXISTS flow.runtime_state;
