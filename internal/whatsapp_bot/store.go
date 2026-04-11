package whatsappbot

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNoSession = errors.New("no whatsapp session found")

type SessionRecord struct {
	WhatsAppJID string
	Email       string
	AccessToken string
	UpdatedAt   time.Time
}

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) SaveSession(ctx context.Context, whatsappJID, email, accessToken string) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO game.whatsapp_sessions (whatsapp_jid, email, access_token, updated_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (whatsapp_jid) DO UPDATE
		SET email = EXCLUDED.email,
		    access_token = EXCLUDED.access_token,
		    updated_at = now()
	`, whatsappJID, email, accessToken)
	return err
}

func (s *Store) GetSession(ctx context.Context, whatsappJID string) (SessionRecord, error) {
	var out SessionRecord
	err := s.db.QueryRow(ctx, `
		SELECT whatsapp_jid, email, access_token, updated_at
		FROM game.whatsapp_sessions
		WHERE whatsapp_jid = $1
	`, whatsappJID).Scan(&out.WhatsAppJID, &out.Email, &out.AccessToken, &out.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionRecord{}, ErrNoSession
	}
	return out, err
}

func (s *Store) DeleteSession(ctx context.Context, whatsappJID string) error {
	_, err := s.db.Exec(ctx, `
		DELETE FROM game.whatsapp_sessions
		WHERE whatsapp_jid = $1
	`, whatsappJID)
	return err
}
