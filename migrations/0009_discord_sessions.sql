CREATE TABLE IF NOT EXISTS game.discord_sessions (
    discord_user_id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    access_token TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_discord_sessions_updated_at
ON game.discord_sessions (updated_at DESC);
