package discordbot

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRecord struct {
	DiscordUserID string
	Email         string
	AccessToken   string
	UpdatedAt     time.Time
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) SaveSession(ctx context.Context, discordUserID, email, accessToken string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO game.discord_sessions (discord_user_id, email, access_token, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (discord_user_id) DO UPDATE
		SET email = EXCLUDED.email,
		    access_token = EXCLUDED.access_token,
		    updated_at = now()
	`, discordUserID, email, accessToken)
	return err
}

func (s *Store) GetSession(ctx context.Context, discordUserID string) (SessionRecord, error) {
	var out SessionRecord
	err := s.db.QueryRow(ctx, `
		SELECT discord_user_id, email, access_token, updated_at
		FROM game.discord_sessions
		WHERE discord_user_id = $1
	`, discordUserID).Scan(&out.DiscordUserID, &out.Email, &out.AccessToken, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionRecord{}, ErrNoSession
	}
	return out, err
}

func (s *Store) DeleteSession(ctx context.Context, discordUserID string) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM game.discord_sessions
		WHERE discord_user_id = $1
	`, discordUserID)
	return err
}
