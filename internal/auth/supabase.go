package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Client struct {
	db *pgxpool.Pool
}

type Session struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	User         User   `json:"user"`
}

type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func NewClient(db *pgxpool.Pool) *Client {
	return &Client{db: db}
}

func (c *Client) SignUp(ctx context.Context, email, password string) (Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return Session{}, fmt.Errorf("email and password are required")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Session{}, fmt.Errorf("hash password: %w", err)
	}

	// If game data was restored but auth isn't, reuse the existing user_id
	// from `users.profiles` so wallet/business/positions still belong to this account.
	userID := ""
	if err := c.db.QueryRow(ctx, `
		SELECT user_id
		FROM users.profiles
		WHERE email = $1
	`, email).Scan(&userID); err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return Session{}, fmt.Errorf("load profile user_id: %w", err)
		}
	}
	if strings.TrimSpace(userID) == "" {
		userID = uuid.NewString()
	}
	token := uuid.NewString()

	_, err = c.db.Exec(ctx, `
		INSERT INTO auth.users (id, email, password_hash, access_token, created_at, updated_at)
		VALUES ($1, $2, $3, $4, now(), now())
	`, userID, email, string(passwordHash), token)
	if err != nil {
		return Session{}, fmt.Errorf("create user: %w", err)
	}

	return Session{
		AccessToken: token,
		TokenType:   "bearer",
		User: User{
			ID:    userID,
			Email: email,
		},
	}, nil
}

func (c *Client) Login(ctx context.Context, email, password string) (Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return Session{}, fmt.Errorf("email and password are required")
	}

	var userID, passwordHash string
	err := c.db.QueryRow(ctx, `
		SELECT id, password_hash
		FROM auth.users
		WHERE email = $1
	`, email).Scan(&userID, &passwordHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Session{}, fmt.Errorf("invalid credentials")
		}
		return Session{}, fmt.Errorf("load user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return Session{}, fmt.Errorf("invalid credentials")
	}

	token := uuid.NewString()
	if _, err := c.db.Exec(ctx, `
		UPDATE auth.users
		SET access_token = $2, updated_at = now()
		WHERE id = $1
	`, userID, token); err != nil {
		return Session{}, fmt.Errorf("store access token: %w", err)
	}

	return Session{
		AccessToken: token,
		TokenType:   "bearer",
		User: User{
			ID:    userID,
			Email: email,
		},
	}, nil
}

func (c *Client) VerifyAccessToken(ctx context.Context, accessToken string) (User, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return User{}, fmt.Errorf("missing access token")
	}

	var user User
	var updatedAt time.Time
	err := c.db.QueryRow(ctx, `
		SELECT id, email, updated_at
		FROM auth.users
		WHERE access_token = $1
	`, accessToken).Scan(&user.ID, &user.Email, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, fmt.Errorf("invalid token")
		}
		return User{}, fmt.Errorf("verify token: %w", err)
	}
	return user, nil
}
