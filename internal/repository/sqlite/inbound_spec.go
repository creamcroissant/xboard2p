package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/creamcroissant/xboard/internal/repository"
)

type inboundSpecRepo struct {
	db *sql.DB
}

func newInboundSpecRepo(db *sql.DB) *inboundSpecRepo {
	return &inboundSpecRepo{db: db}
}

func (r *inboundSpecRepo) Create(ctx context.Context, spec *repository.InboundSpec) error {
	if spec == nil {
		return errors.New("inbound spec is nil")
	}

	now := time.Now().Unix()
	if spec.CreatedAt == 0 {
		spec.CreatedAt = now
	}
	spec.UpdatedAt = now

	semanticSpec := normalizeJSONObject(spec.SemanticSpec)
	coreSpecific := normalizeJSONObject(spec.CoreSpecific)

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO inbound_specs (
			agent_host_id, core_type, tag, enabled, semantic_spec, core_specific,
			desired_revision, created_by, updated_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		spec.AgentHostID,
		spec.CoreType,
		spec.Tag,
		boolToInt(spec.Enabled),
		string(semanticSpec),
		string(coreSpecific),
		spec.DesiredRevision,
		spec.CreatedBy,
		spec.UpdatedBy,
		spec.CreatedAt,
		spec.UpdatedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	spec.ID = id
	spec.SemanticSpec = semanticSpec
	spec.CoreSpecific = coreSpecific
	return nil
}

func (r *inboundSpecRepo) Update(ctx context.Context, spec *repository.InboundSpec) error {
	if spec == nil {
		return errors.New("inbound spec is nil")
	}

	spec.UpdatedAt = time.Now().Unix()
	semanticSpec := normalizeJSONObject(spec.SemanticSpec)
	coreSpecific := normalizeJSONObject(spec.CoreSpecific)

	_, err := r.db.ExecContext(ctx, `
		UPDATE inbound_specs
		SET agent_host_id = ?, core_type = ?, tag = ?, enabled = ?, semantic_spec = ?,
			core_specific = ?, desired_revision = ?, created_by = ?, updated_by = ?, updated_at = ?
		WHERE id = ?
	`,
		spec.AgentHostID,
		spec.CoreType,
		spec.Tag,
		boolToInt(spec.Enabled),
		string(semanticSpec),
		string(coreSpecific),
		spec.DesiredRevision,
		spec.CreatedBy,
		spec.UpdatedBy,
		spec.UpdatedAt,
		spec.ID,
	)
	if err != nil {
		return err
	}

	spec.SemanticSpec = semanticSpec
	spec.CoreSpecific = coreSpecific
	return nil
}

func (r *inboundSpecRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM inbound_specs WHERE id = ?`, id)
	return err
}

func (r *inboundSpecRepo) FindByID(ctx context.Context, id int64) (*repository.InboundSpec, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, agent_host_id, core_type, tag, enabled, semantic_spec, core_specific,
			desired_revision, created_by, updated_by, created_at, updated_at
		FROM inbound_specs
		WHERE id = ?
	`, id)

	return r.scanInboundSpec(row)
}

func (r *inboundSpecRepo) FindByHostCoreTag(ctx context.Context, agentHostID int64, coreType, tag string) (*repository.InboundSpec, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT
			id, agent_host_id, core_type, tag, enabled, semantic_spec, core_specific,
			desired_revision, created_by, updated_by, created_at, updated_at
		FROM inbound_specs
		WHERE agent_host_id = ? AND core_type = ? AND tag = ?
		LIMIT 1
	`, agentHostID, coreType, tag)

	return r.scanInboundSpec(row)
}

func (r *inboundSpecRepo) List(ctx context.Context, filter repository.InboundSpecFilter) ([]*repository.InboundSpec, error) {
	query := strings.Builder{}
	args := make([]any, 0, 8)

	query.WriteString(`
		SELECT
			id, agent_host_id, core_type, tag, enabled, semantic_spec, core_specific,
			desired_revision, created_by, updated_by, created_at, updated_at
		FROM inbound_specs
		WHERE 1 = 1
	`)

	if filter.AgentHostID != nil {
		query.WriteString(" AND agent_host_id = ?")
		args = append(args, *filter.AgentHostID)
	}
	if filter.CoreType != nil {
		query.WriteString(" AND core_type = ?")
		args = append(args, *filter.CoreType)
	}
	if filter.Tag != nil {
		query.WriteString(" AND tag = ?")
		args = append(args, *filter.Tag)
	}
	if filter.Enabled != nil {
		query.WriteString(" AND enabled = ?")
		args = append(args, boolToInt(*filter.Enabled))
	}

	limit, offset := normalizePagination(filter.Limit, filter.Offset, 100)
	query.WriteString(" ORDER BY updated_at DESC, id DESC LIMIT ? OFFSET ?")
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var specs []*repository.InboundSpec
	for rows.Next() {
		spec, err := r.scanInboundSpec(rows)
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}
	return specs, rows.Err()
}

func (r *inboundSpecRepo) Count(ctx context.Context, filter repository.InboundSpecFilter) (int64, error) {
	query := strings.Builder{}
	args := make([]any, 0, 6)

	query.WriteString(`SELECT COUNT(*) FROM inbound_specs WHERE 1 = 1`)

	if filter.AgentHostID != nil {
		query.WriteString(" AND agent_host_id = ?")
		args = append(args, *filter.AgentHostID)
	}
	if filter.CoreType != nil {
		query.WriteString(" AND core_type = ?")
		args = append(args, *filter.CoreType)
	}
	if filter.Tag != nil {
		query.WriteString(" AND tag = ?")
		args = append(args, *filter.Tag)
	}
	if filter.Enabled != nil {
		query.WriteString(" AND enabled = ?")
		args = append(args, boolToInt(*filter.Enabled))
	}

	var total int64
	err := r.db.QueryRowContext(ctx, query.String(), args...).Scan(&total)
	return total, err
}

type inboundSpecScanner interface {
	Scan(dest ...any) error
}

func (r *inboundSpecRepo) scanInboundSpec(scanner inboundSpecScanner) (*repository.InboundSpec, error) {
	var spec repository.InboundSpec
	var enabled int
	var semanticSpec sql.NullString
	var coreSpecific sql.NullString

	err := scanner.Scan(
		&spec.ID,
		&spec.AgentHostID,
		&spec.CoreType,
		&spec.Tag,
		&enabled,
		&semanticSpec,
		&coreSpecific,
		&spec.DesiredRevision,
		&spec.CreatedBy,
		&spec.UpdatedBy,
		&spec.CreatedAt,
		&spec.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	spec.Enabled = enabled != 0
	spec.SemanticSpec = parseJSONObject(semanticSpec)
	spec.CoreSpecific = parseJSONObject(coreSpecific)

	return &spec, nil
}

func normalizeJSONObject(raw []byte) []byte {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return []byte("{}")
	}
	return raw
}

func parseJSONObject(value sql.NullString) []byte {
	if !value.Valid || len(strings.TrimSpace(value.String)) == 0 {
		return []byte("{}")
	}
	return []byte(value.String)
}
