package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type Repository interface {
	Create(ctx context.Context, log Log) error
	List(ctx context.Context, filter ListFilter) ([]Log, error)
}

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, log Log) error {
	metadata := log.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode audit metadata: %w", err)
	}
	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO audit_logs (
			id, actor_user_id, action, target_type, target_id, ip_addr, user_agent, metadata, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		log.ID,
		log.ActorUserID,
		log.Action,
		log.TargetType,
		log.TargetID,
		log.IPAddr,
		log.UserAgent,
		data,
		log.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create audit log: %w", err)
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter) ([]Log, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	where := []string{"1 = 1"}
	args := []any{}
	add := func(condition string, value any) {
		args = append(args, value)
		where = append(where, fmt.Sprintf(condition, len(args)))
	}
	if filter.ActorUserID != nil && *filter.ActorUserID != "" {
		add("actor_user_id = $%d", *filter.ActorUserID)
	}
	if filter.Action != "" {
		add("action = $%d", filter.Action)
	}
	if filter.TargetType != "" {
		add("target_type = $%d", filter.TargetType)
	}
	if filter.TargetID != "" {
		add("target_id = $%d", filter.TargetID)
	}
	if filter.From != nil {
		add("created_at >= $%d", *filter.From)
	}
	if filter.To != nil {
		add("created_at <= $%d", *filter.To)
	}
	args = append(args, limit, offset)
	query := fmt.Sprintf(
		`SELECT id, actor_user_id, action, target_type, target_id, ip_addr, user_agent, metadata, created_at
		FROM audit_logs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "),
		len(args)-1,
		len(args),
	)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []Log
	for rows.Next() {
		var log Log
		var metadata []byte
		if err := rows.Scan(
			&log.ID,
			&log.ActorUserID,
			&log.Action,
			&log.TargetType,
			&log.TargetID,
			&log.IPAddr,
			&log.UserAgent,
			&metadata,
			&log.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		if len(metadata) > 0 {
			if err := json.Unmarshal(metadata, &log.Metadata); err != nil {
				return nil, fmt.Errorf("decode audit metadata: %w", err)
			}
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit logs: %w", err)
	}
	return logs, nil
}
