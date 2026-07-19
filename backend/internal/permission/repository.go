package permission

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("permission not found")

type DocumentOwnerRepository interface {
	DocumentOwnerID(ctx context.Context, documentID string) (string, error)
}

type Repository interface {
	DocumentOwnerRepository
	FindForUser(ctx context.Context, documentID string, userID string) (Permission, error)
	ListForDocument(ctx context.Context, documentID string) ([]Permission, error)
	Create(ctx context.Context, permission Permission) error
	Delete(ctx context.Context, documentID string, permissionID string) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) DocumentOwnerID(ctx context.Context, documentID string) (string, error) {
	var ownerID string
	err := r.db.QueryRowContext(
		ctx,
		`SELECT owner_id FROM documents WHERE id = $1 AND deleted_at IS NULL`,
		documentID,
	).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("find document owner: %w", err)
	}
	return ownerID, nil
}

func (r *PostgresRepository) FindForUser(ctx context.Context, documentID string, userID string) (Permission, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, document_id, subject_type, subject_id, NULL, NULL, permission, created_by, created_at
		FROM document_permissions
		WHERE document_id = $1 AND subject_type = 'user' AND subject_id = $2`,
		documentID,
		userID,
	)
	return scanPermission(row)
}

func (r *PostgresRepository) ListForDocument(ctx context.Context, documentID string) ([]Permission, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT p.id, p.document_id, p.subject_type, p.subject_id, u.display_name, u.email,
			p.permission, p.created_by, p.created_at
		FROM document_permissions p
		LEFT JOIN users u ON p.subject_type = 'user' AND p.subject_id = u.id
		WHERE p.document_id = $1
		ORDER BY p.created_at ASC`,
		documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list document permissions: %w", err)
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		permission, err := scanPermission(rows)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate document permissions: %w", err)
	}
	return permissions, nil
}

func (r *PostgresRepository) Create(ctx context.Context, permission Permission) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO document_permissions (
			id, document_id, subject_type, subject_id, permission, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (document_id, subject_type, subject_id)
		DO UPDATE SET permission = EXCLUDED.permission, created_by = EXCLUDED.created_by, created_at = EXCLUDED.created_at`,
		permission.ID,
		permission.DocumentID,
		permission.SubjectType,
		permission.SubjectID,
		permission.Permission,
		permission.CreatedBy,
		permission.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create document permission: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, documentID string, permissionID string) error {
	result, err := r.db.ExecContext(
		ctx,
		`DELETE FROM document_permissions WHERE document_id = $1 AND id = $2`,
		documentID,
		permissionID,
	)
	if err != nil {
		return fmt.Errorf("delete document permission: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete permission rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPermission(row rowScanner) (Permission, error) {
	var permission Permission
	var subjectDisplayName sql.NullString
	var subjectEmail sql.NullString
	err := row.Scan(
		&permission.ID,
		&permission.DocumentID,
		&permission.SubjectType,
		&permission.SubjectID,
		&subjectDisplayName,
		&subjectEmail,
		&permission.Permission,
		&permission.CreatedBy,
		&permission.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Permission{}, ErrNotFound
	}
	if err != nil {
		return Permission{}, fmt.Errorf("scan document permission: %w", err)
	}
	if subjectDisplayName.Valid {
		permission.SubjectDisplayName = &subjectDisplayName.String
	}
	if subjectEmail.Valid {
		permission.SubjectEmail = &subjectEmail.String
	}
	return permission, nil
}
