package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ChronoCoders/sentra/internal/models"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite" // Register sqlite driver
)

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	if err := initSchema(db); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	// Simple migration
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN role TEXT DEFAULT 'viewer'")
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN password TEXT DEFAULT ''")

	// Ensure admin has a password
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	_, _ = db.Exec("UPDATE users SET password = ? WHERE email = 'admin@sentra.io' AND (password IS NULL OR password = '')", string(hash))

	return &Store{db: db}, nil
}

func initSchema(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS organizations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			name TEXT,
			role TEXT DEFAULT 'viewer',
			password TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(org_id) REFERENCES organizations(id)
		);`,
		`CREATE TABLE IF NOT EXISTS servers (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL,
			hostname TEXT,
			public_key TEXT,
			endpoint TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(org_id) REFERENCES organizations(id)
		);`,
		`CREATE TABLE IF NOT EXISTS peers (
			public_key TEXT PRIMARY KEY,
			endpoint TEXT,
			allowed_ips TEXT,
			latest_handshake DATETIME,
			receive_bytes INTEGER,
			transmit_bytes INTEGER
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}

	// Seed default user
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err == nil && count == 0 {
		// Create default org
		_, _ = db.Exec(`INSERT INTO organizations (id, name) VALUES ('org1', 'Default Org')`)
		// Create default user
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		_, _ = db.Exec(`INSERT INTO users (id, org_id, email, name, role, password) VALUES ('admin', 'org1', 'admin@sentra.io', 'Admin', 'admin', ?)`, string(hash))
	}

	return nil
}

func (s *Store) CreateUser(ctx context.Context, u *models.User) error {
	query := `INSERT INTO users (id, org_id, email, name, role, password, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.ExecContext(ctx, query, u.ID, u.OrgID, u.Email, u.Name, u.Role, u.Password, u.CreatedAt)
	return err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT id, org_id, email, name, role, password, created_at FROM users WHERE email = ?`
	row := s.db.QueryRowContext(ctx, query, email)

	u := &models.User{}

	if err := row.Scan(&u.ID, &u.OrgID, &u.Email, &u.Name, &u.Role, &u.Password, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return u, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}
