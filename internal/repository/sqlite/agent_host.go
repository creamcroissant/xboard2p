package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/creamcroissant/xboard/internal/repository"
)

type agentHostRepo struct {
	db *sql.DB
}

func newAgentHostRepo(db *sql.DB) *agentHostRepo {
	return &agentHostRepo{db: db}
}

func (r *agentHostRepo) Create(ctx context.Context, host *repository.AgentHost) error {
	now := time.Now().Unix()
	host.CreatedAt = now
	host.UpdatedAt = now

	capsJSON, err := json.Marshal(host.Capabilities)
	if err != nil {
		return fmt.Errorf("encode capabilities: %w", err)
	}
	if host.Capabilities == nil {
		capsJSON = []byte("[]")
	}
	tagsJSON, err := json.Marshal(host.BuildTags)
	if err != nil {
		return fmt.Errorf("encode build tags: %w", err)
	}
	if host.BuildTags == nil {
		tagsJSON = []byte("[]")
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO agent_hosts (
			name, host, token, status, template_id, core_version, capabilities, build_tags,
			cpu_total, cpu_used, mem_total, mem_used,
			disk_total, disk_used, upload_total, download_total,
			last_heartbeat_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		host.Name, host.Host, host.Token, host.Status, host.TemplateID,
		host.CoreVersion, string(capsJSON), string(tagsJSON),
		host.CPUTotal, host.CPUUsed, host.MemTotal, host.MemUsed,
		host.DiskTotal, host.DiskUsed, host.UploadTotal, host.DownloadTotal,
		host.LastHeartbeatAt, host.CreatedAt, host.UpdatedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	host.ID = id
	return nil
}

func (r *agentHostRepo) FindByID(ctx context.Context, id int64) (*repository.AgentHost, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, host, token, status, template_id, core_version, capabilities, build_tags,
			cpu_total, cpu_used, mem_total, mem_used,
			disk_total, disk_used, upload_total, download_total,
			last_heartbeat_at, created_at, updated_at
		FROM agent_hosts WHERE id = ?
	`, id)

	return r.scanHost(row)
}

func (r *agentHostRepo) FindByHost(ctx context.Context, host string) (*repository.AgentHost, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, host, token, status, template_id, core_version, capabilities, build_tags,
			cpu_total, cpu_used, mem_total, mem_used,
			disk_total, disk_used, upload_total, download_total,
			last_heartbeat_at, created_at, updated_at
		FROM agent_hosts WHERE host = ?
	`, host)

	return r.scanHost(row)
}

func (r *agentHostRepo) FindByToken(ctx context.Context, token string) (*repository.AgentHost, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, host, token, status, template_id, core_version, capabilities, build_tags,
			cpu_total, cpu_used, mem_total, mem_used,
			disk_total, disk_used, upload_total, download_total,
			last_heartbeat_at, created_at, updated_at
		FROM agent_hosts WHERE token = ?
	`, token)

	return r.scanHost(row)
}

func (r *agentHostRepo) Update(ctx context.Context, host *repository.AgentHost) error {
	host.UpdatedAt = time.Now().Unix()

	capsJSON, err := json.Marshal(host.Capabilities)
	if err != nil {
		return fmt.Errorf("encode capabilities: %w", err)
	}
	if host.Capabilities == nil {
		capsJSON = []byte("[]")
	}
	tagsJSON, err := json.Marshal(host.BuildTags)
	if err != nil {
		return fmt.Errorf("encode build tags: %w", err)
	}
	if host.BuildTags == nil {
		tagsJSON = []byte("[]")
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE agent_hosts SET
			name = ?, host = ?, token = ?, status = ?, template_id = ?,
			core_version = ?, capabilities = ?, build_tags = ?,
			cpu_total = ?, cpu_used = ?, mem_total = ?, mem_used = ?,
			disk_total = ?, disk_used = ?, upload_total = ?, download_total = ?,
			last_heartbeat_at = ?, updated_at = ?
		WHERE id = ?
	`,
		host.Name, host.Host, host.Token, host.Status, host.TemplateID,
		host.CoreVersion, string(capsJSON), string(tagsJSON),
		host.CPUTotal, host.CPUUsed, host.MemTotal, host.MemUsed,
		host.DiskTotal, host.DiskUsed, host.UploadTotal, host.DownloadTotal,
		host.LastHeartbeatAt, host.UpdatedAt, host.ID,
	)
	return err
}

func (r *agentHostRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM agent_hosts WHERE id = ?`, id)
	return err
}

func (r *agentHostRepo) ListAll(ctx context.Context) ([]*repository.AgentHost, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, host, token, status, template_id, core_version, capabilities, build_tags,
			cpu_total, cpu_used, mem_total, mem_used,
			disk_total, disk_used, upload_total, download_total,
			last_heartbeat_at, created_at, updated_at
		FROM agent_hosts ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hosts []*repository.AgentHost
	for rows.Next() {
		host, err := r.scanHostFromRows(rows)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, host)
	}
	return hosts, rows.Err()
}

func (r *agentHostRepo) UpdateStatus(ctx context.Context, id int64, status int, heartbeatAt int64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE agent_hosts SET
			status = ?,
			last_heartbeat_at = ?,
			updated_at = ?
		WHERE id = ?
	`, status, heartbeatAt, time.Now().Unix(), id)
	return err
}

func (r *agentHostRepo) UpdateMetrics(ctx context.Context, id int64, metrics repository.AgentHostMetrics) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE agent_hosts SET
			cpu_total = ?, cpu_used = ?,
			mem_total = ?, mem_used = ?,
			disk_total = ?, disk_used = ?,
			upload_total = ?, download_total = ?,
			last_heartbeat_at = ?,
			status = 1,
			updated_at = ?
		WHERE id = ?
	`,
		metrics.CPUTotal, metrics.CPUUsed,
		metrics.MemTotal, metrics.MemUsed,
		metrics.DiskTotal, metrics.DiskUsed,
		metrics.UploadTotal, metrics.DownloadTotal,
		time.Now().Unix(), time.Now().Unix(), id,
	)
	return err
}

func (r *agentHostRepo) scanHost(row *sql.Row) (*repository.AgentHost, error) {
	var h repository.AgentHost
	var capsJSON, tagsJSON string

	err := row.Scan(
		&h.ID, &h.Name, &h.Host, &h.Token, &h.Status, &h.TemplateID,
		&h.CoreVersion, &capsJSON, &tagsJSON,
		&h.CPUTotal, &h.CPUUsed, &h.MemTotal, &h.MemUsed,
		&h.DiskTotal, &h.DiskUsed, &h.UploadTotal, &h.DownloadTotal,
		&h.LastHeartbeatAt, &h.CreatedAt, &h.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Parse JSON arrays
	if capsJSON != "" {
		if err := json.Unmarshal([]byte(capsJSON), &h.Capabilities); err != nil {
			return nil, fmt.Errorf("decode capabilities: %w", err)
		}
	}
	if h.Capabilities == nil {
		h.Capabilities = []string{}
	}
	if tagsJSON != "" {
		if err := json.Unmarshal([]byte(tagsJSON), &h.BuildTags); err != nil {
			return nil, fmt.Errorf("decode build tags: %w", err)
		}
	}
	if h.BuildTags == nil {
		h.BuildTags = []string{}
	}

	return &h, nil
}

func (r *agentHostRepo) scanHostFromRows(rows *sql.Rows) (*repository.AgentHost, error) {
	var h repository.AgentHost
	var capsJSON, tagsJSON string

	err := rows.Scan(
		&h.ID, &h.Name, &h.Host, &h.Token, &h.Status, &h.TemplateID,
		&h.CoreVersion, &capsJSON, &tagsJSON,
		&h.CPUTotal, &h.CPUUsed, &h.MemTotal, &h.MemUsed,
		&h.DiskTotal, &h.DiskUsed, &h.UploadTotal, &h.DownloadTotal,
		&h.LastHeartbeatAt, &h.CreatedAt, &h.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse JSON arrays
	if capsJSON != "" {
		if err := json.Unmarshal([]byte(capsJSON), &h.Capabilities); err != nil {
			return nil, fmt.Errorf("decode capabilities: %w", err)
		}
	}
	if h.Capabilities == nil {
		h.Capabilities = []string{}
	}
	if tagsJSON != "" {
		if err := json.Unmarshal([]byte(tagsJSON), &h.BuildTags); err != nil {
			return nil, fmt.Errorf("decode build tags: %w", err)
		}
	}
	if h.BuildTags == nil {
		h.BuildTags = []string{}
	}

	return &h, nil
}

// UpdateCapabilities updates agent capabilities.
func (r *agentHostRepo) UpdateCapabilities(ctx context.Context, id int64, coreVersion string, capabilities, buildTags []string) error {
	capsJSON, err := json.Marshal(capabilities)
	if err != nil {
		return fmt.Errorf("encode capabilities: %w", err)
	}
	if capabilities == nil {
		capsJSON = []byte("[]")
	}
	tagsJSON, err := json.Marshal(buildTags)
	if err != nil {
		return fmt.Errorf("encode build tags: %w", err)
	}
	if buildTags == nil {
		tagsJSON = []byte("[]")
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE agent_hosts SET
			core_version = ?, capabilities = ?, build_tags = ?, updated_at = ?
		WHERE id = ?
	`, coreVersion, string(capsJSON), string(tagsJSON), time.Now().Unix(), id)
	return err
}

func (r *agentHostRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM agent_hosts").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *agentHostRepo) CountOnline(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM agent_hosts WHERE status = 1").Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
