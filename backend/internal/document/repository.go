package document

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("document not found")

type Repository interface {
	CreateWithVersion(ctx context.Context, doc Document, version Version) error
	ListByOwner(ctx context.Context, ownerID string) ([]Document, error)
	ListAccessible(ctx context.Context, userID string) ([]Document, error)
	FindByID(ctx context.Context, id string) (Document, error)
	SoftDeleteForOwner(ctx context.Context, id string, ownerID string, deletedAt time.Time) error
	ListVersions(ctx context.Context, documentID string) ([]Version, error)
	HasOnlyOfficeSave(ctx context.Context, documentID string, documentKey string) (bool, error)
	AddVersion(ctx context.Context, documentID string, version Version, documentKey string, updatedAt time.Time) (bool, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateWithVersion(ctx context.Context, doc Document, version Version) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create document tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO documents (
			id, owner_id, title, original_filename, file_ext, mime_type, storage_key,
			current_version_id, size_bytes, deleted_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		doc.ID,
		doc.OwnerID,
		doc.Title,
		doc.OriginalFilename,
		doc.FileExt,
		doc.MimeType,
		doc.StorageKey,
		doc.CurrentVersionID,
		doc.SizeBytes,
		doc.DeletedAt,
		doc.CreatedAt,
		doc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert document: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO document_versions (
			id, document_id, version_no, storage_key, size_bytes, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		version.ID,
		version.DocumentID,
		version.VersionNo,
		version.StorageKey,
		version.SizeBytes,
		version.CreatedBy,
		version.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert document version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create document tx: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListByOwner(ctx context.Context, ownerID string) ([]Document, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, owner_id, title, original_filename, file_ext, mime_type, storage_key,
			current_version_id, size_bytes, deleted_at, created_at, updated_at
		FROM documents
		WHERE owner_id = $1 AND deleted_at IS NULL
		ORDER BY updated_at DESC`,
		ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		doc, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate documents: %w", err)
	}
	return docs, nil
}

func (r *PostgresRepository) ListAccessible(ctx context.Context, userID string) ([]Document, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT DISTINCT d.id, d.owner_id, d.title, d.original_filename, d.file_ext, d.mime_type, d.storage_key,
			d.current_version_id, d.size_bytes, d.deleted_at, d.created_at, d.updated_at
		FROM documents d
		LEFT JOIN document_permissions p
			ON p.document_id = d.id AND p.subject_type = 'user' AND p.subject_id = $1
		WHERE d.deleted_at IS NULL AND (d.owner_id = $1 OR p.permission IN ('viewer', 'editor'))
		ORDER BY d.updated_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list accessible documents: %w", err)
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		doc, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate accessible documents: %w", err)
	}
	return docs, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, id string) (Document, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, owner_id, title, original_filename, file_ext, mime_type, storage_key,
			current_version_id, size_bytes, deleted_at, created_at, updated_at
		FROM documents
		WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	return scanDocument(row)
}

func (r *PostgresRepository) SoftDeleteForOwner(ctx context.Context, id string, ownerID string, deletedAt time.Time) error {
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE documents SET deleted_at = $1, updated_at = $1
		WHERE id = $2 AND owner_id = $3 AND deleted_at IS NULL`,
		deletedAt,
		id,
		ownerID,
	)
	if err != nil {
		return fmt.Errorf("soft delete document: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("soft delete rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) ListVersions(ctx context.Context, documentID string) ([]Version, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT v.id, v.document_id, v.version_no, v.storage_key, v.size_bytes, v.created_by, v.created_at
		FROM document_versions v
		JOIN documents d ON d.id = v.document_id
		WHERE d.id = $1 AND d.deleted_at IS NULL
		ORDER BY v.version_no DESC`,
		documentID,
	)
	if err != nil {
		return nil, fmt.Errorf("list document versions: %w", err)
	}
	defer rows.Close()

	var versions []Version
	for rows.Next() {
		version, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate document versions: %w", err)
	}
	return versions, nil
}

func (r *PostgresRepository) AddVersion(ctx context.Context, documentID string, version Version, documentKey string, updatedAt time.Time) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin add version tx: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(
		ctx,
		`INSERT INTO onlyoffice_saves (document_id, document_key, version_id, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (document_id, document_key) DO NOTHING`,
		documentID,
		documentKey,
		version.ID,
		updatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("record onlyoffice save: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("record onlyoffice save rows affected: %w", err)
	}
	if rows == 0 {
		return false, nil
	}

	var versionNo int64
	err = tx.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(version_no), 0) + 1 FROM document_versions WHERE document_id = $1`,
		documentID,
	).Scan(&versionNo)
	if err != nil {
		return false, fmt.Errorf("next document version: %w", err)
	}
	version.VersionNo = versionNo

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO document_versions (
			id, document_id, version_no, storage_key, size_bytes, created_by, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		version.ID,
		version.DocumentID,
		version.VersionNo,
		version.StorageKey,
		version.SizeBytes,
		version.CreatedBy,
		version.CreatedAt,
	)
	if err != nil {
		return false, fmt.Errorf("insert onlyoffice version: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE documents
		SET current_version_id = $1, storage_key = $2, size_bytes = $3, updated_at = $4
		WHERE id = $5 AND deleted_at IS NULL`,
		version.ID,
		version.StorageKey,
		version.SizeBytes,
		updatedAt,
		documentID,
	)
	if err != nil {
		return false, fmt.Errorf("update document current version: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE onlyoffice_saves SET version_id = $1 WHERE document_id = $2 AND document_key = $3`,
		version.ID,
		documentID,
		documentKey,
	)
	if err != nil {
		return false, fmt.Errorf("update onlyoffice save version: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit add version tx: %w", err)
	}
	return true, nil
}

func (r *PostgresRepository) HasOnlyOfficeSave(ctx context.Context, documentID string, documentKey string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1 FROM onlyoffice_saves WHERE document_id = $1 AND document_key = $2
		)`,
		documentID,
		documentKey,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check onlyoffice save: %w", err)
	}
	return exists, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDocument(row rowScanner) (Document, error) {
	var doc Document
	err := row.Scan(
		&doc.ID,
		&doc.OwnerID,
		&doc.Title,
		&doc.OriginalFilename,
		&doc.FileExt,
		&doc.MimeType,
		&doc.StorageKey,
		&doc.CurrentVersionID,
		&doc.SizeBytes,
		&doc.DeletedAt,
		&doc.CreatedAt,
		&doc.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Document{}, ErrNotFound
	}
	if err != nil {
		return Document{}, fmt.Errorf("scan document: %w", err)
	}
	return doc, nil
}

func scanVersion(row rowScanner) (Version, error) {
	var version Version
	err := row.Scan(
		&version.ID,
		&version.DocumentID,
		&version.VersionNo,
		&version.StorageKey,
		&version.SizeBytes,
		&version.CreatedBy,
		&version.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Version{}, ErrNotFound
	}
	if err != nil {
		return Version{}, fmt.Errorf("scan document version: %w", err)
	}
	return version, nil
}
