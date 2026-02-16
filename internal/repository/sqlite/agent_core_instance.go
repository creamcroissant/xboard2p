package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/creamcroissant/xboard/internal/repository"
)

type agentCoreInstanceRepo struct {
	db *sql.DB
}

func newAgentCoreInstanceRepo(db *sql.DB) *agentCoreInstanceRepo {
	return &agentCoreInstanceRepo{db: db}
}

func (r *agentCoreInstanceRepo) Create(ctx context.Context, instance *repository.AgentCoreInstance) error {
	if instance == nil {
		return errors.New("instance is nil")
	}

	now := time.Now().Unix()
	instance.CreatedAt = now
	instance.UpdatedAt = now

	portsJSON, err := json.Marshal(instance.ListenPorts)
	if err != nil {
		return err
	}
	if instance.ListenPorts == nil {
		portsJSON = []byte("[]")
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_core_instances (
			agent_host_id, instance_id, core_type, status, listen_ports,
			config_template_id, config_hash, started_at, last_heartbeat_at,
			error_message, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		instance.AgentHostID,
		instance.InstanceID,
		instance.CoreType,
		instance.Status,
		string(portsJSON),
		optionalInt64(instance.ConfigTemplateID),
		instance.ConfigHash,
		optionalInt64(instance.StartedAt),
		optionalInt64(instance.LastHeartbeatAt),
		instance.ErrorMessage,
		instance.CreatedAt,
		instance.UpdatedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	instance.ID = id
	return nil
}

func (r *agentCoreInstanceRepo) Update(ctx context.Context, instance *repository.AgentCoreInstance) error {
	if instance == nil {
		return errors.New("instance is nil")
	}

	instance.UpdatedAt = time.Now().Unix()

	portsJSON, err := json.Marshal(instance.ListenPorts)
	if err != nil {
		return err
	}
	if instance.ListenPorts == nil {
		portsJSON = []byte("[]")
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE agent_core_instances SET
			agent_host_id = ?, instance_id = ?, core_type = ?, status = ?, listen_ports = ?,
			config_template_id = ?, config_hash = ?, started_at = ?, last_heartbeat_at = ?,
			error_message = ?, updated_at = ?
		WHERE id = ?
	`,
		instance.AgentHostID,
		instance.InstanceID,
		instance.CoreType,
		instance.Status,
		string(portsJSON),
		optionalInt64(instance.ConfigTemplateID),
		instance.ConfigHash,
		optionalInt64(instance.StartedAt),
		optionalInt64(instance.LastHeartbeatAt),
		instance.ErrorMessage,
		instance.UpdatedAt,
		instance.ID,
	)
	return err
}

func (r *agentCoreInstanceRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM agent_core_instances WHERE id = ?`, id)
	return err
}

func (r *agentCoreInstanceRepo) FindByID(ctx context.Context, id int64) (*repository.AgentCoreInstance, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, agent_host_id, instance_id, core_type, status, listen_ports,
			config_template_id, config_hash, started_at, last_heartbeat_at,
			error_message, created_at, updated_at
		FROM agent_core_instances WHERE id = ?
	`, id)

	return r.scanInstance(row)
}

func (r *agentCoreInstanceRepo) FindByInstanceID(ctx context.Context, agentHostID int64, instanceID string) (*repository.AgentCoreInstance, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, agent_host_id, instance_id, core_type, status, listen_ports,
			config_template_id, config_hash, started_at, last_heartbeat_at,
			error_message, created_at, updated_at
		FROM agent_core_instances WHERE agent_host_id = ? AND instance_id = ?
		LIMIT 1
	`, agentHostID, instanceID)

	return r.scanInstance(row)
}

func (r *agentCoreInstanceRepo) ListByAgentHostID(ctx context.Context, agentHostID int64) ([]*repository.AgentCoreInstance, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, agent_host_id, instance_id, core_type, status, listen_ports,
			config_template_id, config_hash, started_at, last_heartbeat_at,
			error_message, created_at, updated_at
		FROM agent_core_instances
		WHERE agent_host_id = ?
		ORDER BY id DESC
	`, agentHostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []*repository.AgentCoreInstance
	for rows.Next() {
		instance, err := r.scanInstanceRow(rows)
		if err != nil {
			return nil, err
		}
		instances = append(instances, instance)
	}
	return instances, rows.Err()
}

func (r *agentCoreInstanceRepo) UpdateHeartbeat(ctx context.Context, agentHostID int64, instanceID string, heartbeatAt int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE agent_core_instances
		SET last_heartbeat_at = ?, updated_at = ?
		WHERE agent_host_id = ? AND instance_id = ?
	`, heartbeatAt, time.Now().Unix(), agentHostID, instanceID)
	return err
}

type agentCoreInstanceScanner interface {
	Scan(dest ...any) error
}

func (r *agentCoreInstanceRepo) scanInstance(scanner agentCoreInstanceScanner) (*repository.AgentCoreInstance, error) {
	var instance repository.AgentCoreInstance
	var listenPorts sql.NullString
	var templateID sql.NullInt64
	var startedAt sql.NullInt64
	var heartbeatAt sql.NullInt64
	var errorMessage sql.NullString

	err := scanner.Scan(
		&instance.ID,
		&instance.AgentHostID,
		&instance.InstanceID,
		&instance.CoreType,
		&instance.Status,
		&listenPorts,
		&templateID,
		&instance.ConfigHash,
		&startedAt,
		&heartbeatAt,
		&errorMessage,
		&instance.CreatedAt,
		&instance.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if listenPorts.Valid {
		if err := json.Unmarshal([]byte(listenPorts.String), &instance.ListenPorts); err != nil {
			return nil, fmt.Errorf("decode listen ports: %w", err)
		}
	}
	if instance.ListenPorts == nil {
		instance.ListenPorts = []int{}
	}
	if templateID.Valid {
		instance.ConfigTemplateID = nullableIntPtr(templateID)
	}
	if startedAt.Valid {
		instance.StartedAt = nullableIntPtr(startedAt)
	}
	if heartbeatAt.Valid {
		instance.LastHeartbeatAt = nullableIntPtr(heartbeatAt)
	}
	if errorMessage.Valid {
		instance.ErrorMessage = errorMessage.String
	}

	return &instance, nil
}

func (r *agentCoreInstanceRepo) scanInstanceRow(rows *sql.Rows) (*repository.AgentCoreInstance, error) {
	var instance repository.AgentCoreInstance
	var listenPorts sql.NullString
	var templateID sql.NullInt64
	var startedAt sql.NullInt64
	var heartbeatAt sql.NullInt64
	var errorMessage sql.NullString

	if err := rows.Scan(
		&instance.ID,
		&instance.AgentHostID,
		&instance.InstanceID,
		&instance.CoreType,
		&instance.Status,
		&listenPorts,
		&templateID,
		&instance.ConfigHash,
		&startedAt,
		&heartbeatAt,
		&errorMessage,
		&instance.CreatedAt,
		&instance.UpdatedAt,
	); err != nil {
		return nil, err
	}

	if listenPorts.Valid {
		if err := json.Unmarshal([]byte(listenPorts.String), &instance.ListenPorts); err != nil {
			return nil, fmt.Errorf("decode listen ports: %w", err)
		}
	}
	if instance.ListenPorts == nil {
		instance.ListenPorts = []int{}
	}
	if templateID.Valid {
		instance.ConfigTemplateID = nullableIntPtr(templateID)
	}
	if startedAt.Valid {
		instance.StartedAt = nullableIntPtr(startedAt)
	}
	if heartbeatAt.Valid {
		instance.LastHeartbeatAt = nullableIntPtr(heartbeatAt)
	}
	if errorMessage.Valid {
		instance.ErrorMessage = errorMessage.String
	}

	return &instance, nil
}
