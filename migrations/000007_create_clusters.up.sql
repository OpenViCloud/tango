CREATE TABLE IF NOT EXISTS clusters (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    error_msg       TEXT NOT NULL DEFAULT '',
    k8s_version     TEXT NOT NULL DEFAULT 'v1.30',
    pod_cidr        TEXT NOT NULL DEFAULT '192.168.0.0/16',
    kubeconfig_enc  TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cluster_nodes (
    cluster_id  TEXT NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    server_id   TEXT NOT NULL REFERENCES servers(id) ON DELETE CASCADE,
    role        TEXT NOT NULL,
    PRIMARY KEY (cluster_id, server_id)
);
