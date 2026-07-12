package local

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"online-colab-document/backend/internal/auth/session"
	"online-colab-document/backend/internal/user"
)

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrEmailAlreadyUsed = errors.New("email already used")
	ErrSessionNotFound  = errors.New("session not found")
)

type UserRepository interface {
	Create(ctx context.Context, user user.User) error
	FindByEmail(ctx context.Context, email string) (user.User, error)
	FindByID(ctx context.Context, id string) (user.User, error)
	FindByOIDCSubject(ctx context.Context, subject string) (user.User, error)
	SetOIDCSubject(ctx context.Context, id string, subject string) (user.User, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, u user.User) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO users (
			id, email, display_name, password_hash, auth_source, oidc_subject,
			is_admin, disabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		u.ID,
		u.Email,
		u.DisplayName,
		u.PasswordHash,
		u.AuthSource,
		u.OIDCSubject,
		u.IsAdmin,
		u.Disabled,
		u.CreatedAt,
		u.UpdatedAt,
	)
	if isUniqueViolation(err) {
		return ErrEmailAlreadyUsed
	}
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (user.User, error) {
	return r.scanUser(ctx, `SELECT id, email, display_name, password_hash, auth_source, oidc_subject,
		is_admin, disabled, created_at, updated_at FROM users WHERE email = $1`, email)
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (user.User, error) {
	return r.scanUser(ctx, `SELECT id, email, display_name, password_hash, auth_source, oidc_subject,
		is_admin, disabled, created_at, updated_at FROM users WHERE id = $1`, id)
}

func (r *PostgresRepository) FindByOIDCSubject(ctx context.Context, subject string) (user.User, error) {
	return r.scanUser(ctx, `SELECT id, email, display_name, password_hash, auth_source, oidc_subject,
		is_admin, disabled, created_at, updated_at FROM users WHERE oidc_subject = $1`, subject)
}

func (r *PostgresRepository) SetOIDCSubject(ctx context.Context, id string, subject string) (user.User, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET oidc_subject = $1, updated_at = NOW() WHERE id = $2`, subject, id)
	if isUniqueViolation(err) {
		return user.User{}, ErrEmailAlreadyUsed
	}
	if err != nil {
		return user.User{}, fmt.Errorf("set oidc subject: %w", err)
	}
	return r.FindByID(ctx, id)
}

func (r *PostgresRepository) scanUser(ctx context.Context, query string, args ...any) (user.User, error) {
	var u user.User
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&u.ID,
		&u.Email,
		&u.DisplayName,
		&u.PasswordHash,
		&u.AuthSource,
		&u.OIDCSubject,
		&u.IsAdmin,
		&u.Disabled,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return user.User{}, ErrUserNotFound
	}
	if err != nil {
		return user.User{}, fmt.Errorf("find user: %w", err)
	}
	return u, nil
}

func (r *PostgresRepository) CreateSession(ctx context.Context, record session.Record) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO sessions (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		record.ID,
		record.UserID,
		record.TokenHash,
		record.ExpiresAt,
		record.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FindSessionByTokenHash(ctx context.Context, tokenHash string) (session.Record, error) {
	var record session.Record
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, token_hash, expires_at, created_at FROM sessions WHERE token_hash = $1`,
		tokenHash,
	).Scan(&record.ID, &record.UserID, &record.TokenHash, &record.ExpiresAt, &record.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return session.Record{}, ErrSessionNotFound
	}
	if err != nil {
		return session.Record{}, fmt.Errorf("find session: %w", err)
	}
	return record, nil
}

func (r *PostgresRepository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
