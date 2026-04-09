CREATE TABLE IF NOT EXISTS servers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    public_ip   TEXT NOT NULL,
    private_ip  TEXT NOT NULL DEFAULT '',
    ssh_user    TEXT NOT NULL DEFAULT 'root',
    ssh_port    INTEGER NOT NULL DEFAULT 22,
    status      TEXT NOT NULL DEFAULT 'pending',
    error_msg   TEXT NOT NULL DEFAULT '',
    last_ping_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
