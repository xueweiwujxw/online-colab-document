package user

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Search(ctx context.Context, query string, limit, offset int) ([]User, error) {
	query = strings.TrimSpace(query)
	args := []any{limit, offset}
	where := "disabled = false"
	if query != "" {
		args = []any{"%" + strings.ToLower(query) + "%", limit, offset}
		where += " AND (LOWER(email) LIKE $1 OR LOWER(display_name) LIKE $1)"
	}
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, email, display_name, password_hash, auth_source, oidc_subject,
			is_admin, disabled, created_at, updated_at, avatar_key
		FROM users
		WHERE %s
		ORDER BY display_name ASC, email ASC
		LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(
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
			&u.AvatarKey,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search users rows: %w", err)
	}
	return users, nil
}

func (r *PostgresRepository) ListAdmin(ctx context.Context, query string, limit int, offset int) ([]User, error) {
	query = strings.TrimSpace(query)
	args := []any{limit, offset}
	where := "TRUE"
	if query != "" {
		args = []any{"%" + strings.ToLower(query) + "%", limit, offset}
		where = "LOWER(email) LIKE $1 OR LOWER(display_name) LIKE $1"
	}
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT id, email, display_name, password_hash, auth_source, oidc_subject, is_admin, disabled, created_at, updated_at, avatar_key FROM users WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, fmt.Errorf("list admin users: %w", err)
	}
	defer rows.Close()
	items := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &u.AuthSource, &u.OIDCSubject, &u.IsAdmin, &u.Disabled, &u.CreatedAt, &u.UpdatedAt, &u.AvatarKey); err != nil {
			return nil, fmt.Errorf("scan admin user: %w", err)
		}
		items = append(items, u)
	}
	return items, rows.Err()
}
