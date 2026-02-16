// 文件路径: internal/repository/sqlite/user_traffic.go
// 模块说明: UserTrafficRepository 的 SQLite 实现，管理用户流量周期和节点选择
package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/creamcroissant/xboard/internal/repository"
)

type userTrafficRepo struct {
	db *sql.DB
}

func newUserTrafficRepo(db *sql.DB) *userTrafficRepo {
	return &userTrafficRepo{db: db}
}

// AddServerSelection adds a server to user's selection list.
func (r *userTrafficRepo) AddServerSelection(ctx context.Context, userID, serverID int64) error {
	now := time.Now().Unix()
	_, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO user_server_selections (user_id, server_id, created_at)
		VALUES (?, ?, ?)
	`, userID, serverID, now)
	return err
}

// RemoveServerSelection removes a server from user's selection list.
func (r *userTrafficRepo) RemoveServerSelection(ctx context.Context, userID, serverID int64) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM user_server_selections WHERE user_id = ? AND server_id = ?
	`, userID, serverID)
	return err
}

// GetUserServerIDs returns all server IDs selected by a user.
func (r *userTrafficRepo) GetUserServerIDs(ctx context.Context, userID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT server_id FROM user_server_selections WHERE user_id = ?
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ClearUserSelections removes all server selections for a user.
func (r *userTrafficRepo) ClearUserSelections(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM user_server_selections WHERE user_id = ?
	`, userID)
	return err
}

// ReplaceUserSelections atomically replaces all selections for a user.
func (r *userTrafficRepo) ReplaceUserSelections(ctx context.Context, userID int64, serverIDs []int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM user_server_selections WHERE user_id = ?
	`, userID); err != nil {
		return err
	}

	if len(serverIDs) == 0 {
		return tx.Commit()
	}

	now := time.Now().Unix()
	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO user_server_selections (user_id, server_id, created_at)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, serverID := range serverIDs {
		if _, err := stmt.ExecContext(ctx, userID, serverID, now); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetCurrentPeriod returns the current active traffic period for a user.
func (r *userTrafficRepo) GetCurrentPeriod(ctx context.Context, userID int64) (*repository.UserTrafficPeriod, error) {
	now := time.Now().Unix()
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, period_start, period_end, upload_bytes, download_bytes, quota_bytes, exceeded, created_at, updated_at
		FROM user_traffic_periods
		WHERE user_id = ? AND period_start <= ? AND period_end > ?
		ORDER BY period_start DESC
		LIMIT 1
	`, userID, now, now)

	var p repository.UserTrafficPeriod
	var exceeded int
	err := row.Scan(&p.ID, &p.UserID, &p.PeriodStart, &p.PeriodEnd, &p.UploadBytes, &p.DownloadBytes, &p.QuotaBytes, &exceeded, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Exceeded = exceeded == 1
	return &p, nil
}

// CreatePeriod creates a new traffic period for a user.
func (r *userTrafficRepo) CreatePeriod(ctx context.Context, period *repository.UserTrafficPeriod) error {
	now := time.Now().Unix()
	exceeded := 0
	if period.Exceeded {
		exceeded = 1
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO user_traffic_periods (user_id, period_start, period_end, upload_bytes, download_bytes, quota_bytes, exceeded, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, period.UserID, period.PeriodStart, period.PeriodEnd, period.UploadBytes, period.DownloadBytes, period.QuotaBytes, exceeded, now, now)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	period.ID = id
	period.CreatedAt = now
	period.UpdatedAt = now
	return nil
}

// IncrementPeriodTraffic atomically increments the traffic counters for the current period.
func (r *userTrafficRepo) IncrementPeriodTraffic(ctx context.Context, userID int64, uploadDelta, downloadDelta int64) error {
	now := time.Now().Unix()
	result, err := r.db.ExecContext(ctx, `
		UPDATE user_traffic_periods
		SET upload_bytes = upload_bytes + ?,
		    download_bytes = download_bytes + ?,
		    updated_at = ?
		WHERE user_id = ? AND period_start <= ? AND period_end > ?
	`, uploadDelta, downloadDelta, now, userID, now, now)
	if err != nil {
		return err
	}
	// If no rows updated, no current period exists (will be created by service layer)
	_, _ = result.RowsAffected()
	return nil
}

// MarkPeriodExceeded marks a specific period as exceeded.
func (r *userTrafficRepo) MarkPeriodExceeded(ctx context.Context, userID int64, periodStart int64) error {
	now := time.Now().Unix()
	_, err := r.db.ExecContext(ctx, `
		UPDATE user_traffic_periods
		SET exceeded = 1, updated_at = ?
		WHERE user_id = ? AND period_start = ?
	`, now, userID, periodStart)
	return err
}

// GetExpiredPeriodUserIDs returns user IDs whose current period has ended.
func (r *userTrafficRepo) GetExpiredPeriodUserIDs(ctx context.Context, nowUnix int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT user_id FROM user_traffic_periods
		WHERE period_end <= ? AND exceeded = 0
	`, nowUnix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetExceededUserIDs returns all user IDs who have exceeded their traffic quota in current period.
func (r *userTrafficRepo) GetExceededUserIDs(ctx context.Context) ([]int64, error) {
	now := time.Now().Unix()
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id FROM user_traffic_periods
		WHERE period_start <= ? AND period_end > ? AND exceeded = 1
	`, now, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetUserTrafficStats returns traffic statistics for a user.
func (r *userTrafficRepo) GetUserTrafficStats(ctx context.Context, userID int64) (*repository.UserTrafficStats, error) {
	period, err := r.GetCurrentPeriod(ctx, userID)
	if err != nil {
		return nil, err
	}
	if period == nil {
		return nil, nil
	}

	total := period.UploadBytes + period.DownloadBytes
	var usedPercent float64
	if period.QuotaBytes > 0 {
		usedPercent = float64(total) / float64(period.QuotaBytes) * 100
	}

	return &repository.UserTrafficStats{
		PeriodStart:   period.PeriodStart,
		PeriodEnd:     period.PeriodEnd,
		UploadBytes:   period.UploadBytes,
		DownloadBytes: period.DownloadBytes,
		TotalBytes:    total,
		QuotaBytes:    period.QuotaBytes,
		UsedPercent:   usedPercent,
		Exceeded:      period.Exceeded,
	}, nil
}
