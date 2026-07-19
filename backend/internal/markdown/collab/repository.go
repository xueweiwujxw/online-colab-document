package collab

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("markdown collab record not found")

type Repository interface {
	LatestSnapshot(ctx context.Context, documentID string) (Snapshot, error)
	ListUpdatesAfter(ctx context.Context, documentID string, updateSeq int64) ([]Update, error)
	NextUpdateSeq(ctx context.Context, documentID string) (int64, error)
	CreateUpdate(ctx context.Context, update Update) error
	CreateSnapshot(ctx context.Context, snapshot Snapshot) error
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) LatestSnapshot(ctx context.Context, documentID string) (Snapshot, error) {
	var snapshot Snapshot
	err := r.db.QueryRowContext(
		ctx,
		`SELECT id, document_id, version_no, content, created_by, created_at
		FROM markdown_snapshots
		WHERE document_id = $1
		ORDER BY version_no DESC
		LIMIT 1`,
		documentID,
	).Scan(
		&snapshot.ID,
		&snapshot.DocumentID,
		&snapshot.VersionNo,
		&snapshot.Content,
		&snapshot.CreatedBy,
		&snapshot.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Snapshot{}, ErrNotFound
	}
	if err != nil {
		return Snapshot{}, fmt.Errorf("latest markdown snapshot: %w", err)
	}
	return snapshot, nil
}

func (r *PostgresRepository) ListUpdatesAfter(ctx context.Context, documentID string, updateSeq int64) ([]Update, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, document_id, update_seq, update_data, created_by, created_at
		FROM markdown_updates
		WHERE document_id = $1 AND update_seq > $2
		ORDER BY update_seq ASC`,
		documentID,
		updateSeq,
	)
	if err != nil {
		return nil, fmt.Errorf("list markdown updates: %w", err)
	}
	defer rows.Close()

	var updates []Update
	for rows.Next() {
		var update Update
		if err := rows.Scan(
			&update.ID,
			&update.DocumentID,
			&update.UpdateSeq,
			&update.UpdateData,
			&update.CreatedBy,
			&update.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan markdown update: %w", err)
		}
		updates = append(updates, update)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate markdown updates: %w", err)
	}
	return updates, nil
}

func (r *PostgresRepository) NextUpdateSeq(ctx context.Context, documentID string) (int64, error) {
	var next int64
	err := r.db.QueryRowContext(
		ctx,
		`SELECT COALESCE(MAX(update_seq), 0) + 1 FROM markdown_updates WHERE document_id = $1`,
		documentID,
	).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("next markdown update seq: %w", err)
	}
	return next, nil
}

func (r *PostgresRepository) CreateUpdate(ctx context.Context, update Update) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO markdown_updates (id, document_id, update_seq, update_data, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		update.ID,
		update.DocumentID,
		update.UpdateSeq,
		update.UpdateData,
		update.CreatedBy,
		update.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create markdown update: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateSnapshot(ctx context.Context, snapshot Snapshot) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO markdown_snapshots (id, document_id, version_no, content, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		snapshot.ID,
		snapshot.DocumentID,
		snapshot.VersionNo,
		snapshot.Content,
		snapshot.CreatedBy,
		snapshot.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create markdown snapshot: %w", err)
	}
	return nil
}
