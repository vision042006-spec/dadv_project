package db

import (
	"context"
	"time"
)

type User struct {
	ID           int64     `json:"id"`
	Email        string   `json:"email"`
	PasswordHash string   `json:"-"`
	GoogleID    string   `json:"google_id,omitempty"`
	Name        string   `json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (db *DB) InitUsersTable(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT,
		google_id TEXT UNIQUE,
		name TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);
	`
	_, err := db.conn.ExecContext(ctx, schema)
	return err
}

func (db *DB) CreateUser(ctx context.Context, email, passwordHash, name string) (int64, error) {
	query := `INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)`
	result, err := db.conn.ExecContext(ctx, query, email, passwordHash, name)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (db *DB) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `SELECT id, email, password_hash, google_id, name, created_at, updated_at FROM users WHERE email = ?`
	user := &User{}
	err := db.conn.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.GoogleID, &user.Name, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (db *DB) GetUserByID(ctx context.Context, id int64) (*User, error) {
	query := `SELECT id, email, password_hash, google_id, name, created_at, updated_at FROM users WHERE id = ?`
	user := &User{}
	err := db.conn.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.GoogleID, &user.Name, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (db *DB) GetUserByGoogleID(ctx context.Context, googleID string) (*User, error) {
	query := `SELECT id, email, password_hash, google_id, name, created_at, updated_at FROM users WHERE google_id = ?`
	user := &User{}
	err := db.conn.QueryRowContext(ctx, query, googleID).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.GoogleID, &user.Name, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (db *DB) UpdateUserPassword(ctx context.Context, userID int64, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.ExecContext(ctx, query, passwordHash, userID)
	return err
}

func (db *DB) UpdateUserName(ctx context.Context, userID int64, name string) error {
	query := `UPDATE users SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.ExecContext(ctx, query, name, userID)
	return err
}

func (db *DB) LinkGoogleAccount(ctx context.Context, userID int64, googleID, email string) error {
	query := `UPDATE users SET google_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.ExecContext(ctx, query, googleID, userID)
	return err
}

func (db *DB) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT COUNT(*) > 0 FROM users WHERE email = ?`
	var exists bool
	err := db.conn.QueryRowContext(ctx, query, email).Scan(&exists)
	return exists, err
}

func (db *DB) GoogleIDExists(ctx context.Context, googleID string) (bool, error) {
	query := `SELECT COUNT(*) > 0 FROM users WHERE google_id = ?`
	var exists bool
	err := db.conn.QueryRowContext(ctx, query, googleID).Scan(&exists)
	return exists, err
}