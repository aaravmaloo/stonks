CREATE TABLE IF NOT EXISTS game.whatsapp_sessions (
    whatsapp_jid VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    access_token TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
