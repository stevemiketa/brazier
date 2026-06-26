package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

const sessionTokenLen = 32 // bytes → 64 hex chars
const sessionTTL = 24 * time.Hour

// Session is the persisted record of an authenticated session.
type Session struct {
	Token     string
	UserID    string
	Login     string // GitHub login
	ExpiresAt time.Time
}

// SessionStore manages session tokens in the database.
type SessionStore struct {
	db *sql.DB
}

// NewSessionStore initialises the sessions table and returns a store.
func NewSessionStore(db *sql.DB) (*SessionStore, error) {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS sessions (
  token      TEXT PRIMARY KEY,
  user_id    TEXT NOT NULL,
  login      TEXT NOT NULL,
  expires_at INTEGER NOT NULL
)`)
	if err != nil {
		return nil, fmt.Errorf("migrate sessions: %w", err)
	}
	return &SessionStore{db: db}, nil
}

// Create issues a new session token for userID/login, valid for sessionTTL.
func (s *SessionStore) Create(ctx context.Context, userID, login string) (Session, error) {
	buf := make([]byte, sessionTokenLen)
	if _, err := rand.Read(buf); err != nil {
		return Session{}, fmt.Errorf("generate token: %w", err)
	}
	token := hex.EncodeToString(buf)
	expiresAt := time.Now().Add(sessionTTL)

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO sessions (token, user_id, login, expires_at) VALUES (?,?,?,?)`,
		token, userID, login, expiresAt.UnixMilli(),
	)
	if err != nil {
		return Session{}, fmt.Errorf("store session: %w", err)
	}
	return Session{Token: token, UserID: userID, Login: login, ExpiresAt: expiresAt}, nil
}

// Validate returns the Session for token if it exists and has not expired.
func (s *SessionStore) Validate(ctx context.Context, token string) (Session, error) {
	var sess Session
	var expiresMs int64
	err := s.db.QueryRowContext(ctx,
		`SELECT token, user_id, login, expires_at FROM sessions WHERE token = ?`, token,
	).Scan(&sess.Token, &sess.UserID, &sess.Login, &expiresMs)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, errors.New("session not found")
	}
	if err != nil {
		return Session{}, err
	}
	sess.ExpiresAt = time.UnixMilli(expiresMs)
	if time.Now().After(sess.ExpiresAt) {
		_ = s.Delete(ctx, token)
		return Session{}, errors.New("session expired")
	}
	return sess, nil
}

// Delete revokes a session token.
func (s *SessionStore) Delete(ctx context.Context, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token = ?`, token)
	return err
}

// DeleteExpired removes all expired sessions from the database.
func (s *SessionStore) DeleteExpired(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE expires_at < ?`, time.Now().UnixMilli())
	return err
}
