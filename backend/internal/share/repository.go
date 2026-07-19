package share

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("share link not found")

type Repository interface {
	Create(ctx context.Context, link Link) error
	ListForDocument(ctx context.Context, documentID string) ([]Link, error)
	FindByID(ctx context.Context, id string) (Link, error)
	FindByTokenHash(ctx context.Context, tokenHash string) (Link, error)
	Disable(ctx context.Context, id string) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, link Link) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO share_links (
			id, document_id, token_hash, permission, expires_at, disabled, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		link.ID,
		link.DocumentID,
		link.TokenHash,
		link.Permission,
		link.ExpiresAt,
		link.Disabled,
		link.CreatedBy,
		link.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create share link: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListForDocument(ctx context.Context, documentID string) ([]Link, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, document_id, token_hash, permission, expires_at, disabled, created_by, created_at
		FROM share_links
		WHERE document_id = $1
		ORDER BY created_at DESC`,
		documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list share links: %w", err)
	}
	defer rows.Close()

	var links []Link
	for rows.Next() {
		link, err := scanLink(rows)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate share links: %w", err)
	}
	return links, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (Link, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, document_id, token_hash, permission, expires_at, disabled, created_by, created_at
		FROM share_links
		WHERE id = $1`,
		id,
	)
	return scanLink(row)
}

func (r *PostgresRepository) FindByTokenHash(ctx context.Context, tokenHash string) (Link, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, document_id, token_hash, permission, expires_at, disabled, created_by, created_at
		FROM share_links
		WHERE token_hash = $1`,
		tokenHash,
	)
	return scanLink(row)
}

func (r *PostgresRepository) Disable(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE share_links SET disabled = TRUE WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("disable share link: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("disable share link rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanLink(row rowScanner) (Link, error) {
	var link Link
	err := row.Scan(
		&link.ID,
		&link.DocumentID,
		&link.TokenHash,
		&link.Permission,
		&link.ExpiresAt,
		&link.Disabled,
		&link.CreatedBy,
		&link.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Link{}, ErrNotFound
	}
	if err != nil {
		return Link{}, fmt.Errorf("scan share link: %w", err)
	}
	return link, nil
}
