package db

import (
	"context"
	"time"
)

func (db *DB) CreatePasswordResetToken(ctx context.Context, email, token string, expires time.Time) error {
	db.PasswordResetTokens[email] = struct {
		Token   string
		Expires time.Time
	}{
		Token:   token,
		Expires: expires,
	}
	return nil
}

func (db *DB) GetPasswordResetToken(ctx context.Context, email string) (string, time.Time, bool) {
	if token, ok := db.PasswordResetTokens[email]; ok {
		return token.Token, token.Expires, true
	}
	return "", time.Time{}, false
}

func (db *DB) DeletePasswordResetToken(ctx context.Context, email string) {
	delete(db.PasswordResetTokens, email)
}