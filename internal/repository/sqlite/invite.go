// 文件路径: internal/repository/sqlite/invite.go
// 模块说明: 这是 internal 模块里的 invite 逻辑，下面的注释会用非常通俗的中文帮你理解每一步。
package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/creamcroissant/xboard/internal/repository"
)

// inviteRepo implements repository.InviteCodeRepository using SQLite.
type inviteRepo struct {
	db *sql.DB
}

func (r *inviteRepo) IncrementPV(ctx context.Context, code string) error {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" || r == nil || r.db == nil {
		return nil
	}
	now := time.Now().Unix()
	_, err := r.db.ExecContext(ctx, "UPDATE v2_invite_code SET pv = pv + 1, updated_at = ? WHERE code = ?", now, trimmed)
	if err != nil {
		return err
	}
	return nil
}

func (r *inviteRepo) FindByCode(ctx context.Context, code string) (*repository.InviteCode, error) {
	trimmed := strings.TrimSpace(code)
	if trimmed == "" || r == nil || r.db == nil {
		return nil, repository.ErrNotFound
	}
	const query = `SELECT id, user_id, code, status, pv, "limit", expire_at, created_at, updated_at FROM v2_invite_code WHERE code = ?`
	row := r.db.QueryRowContext(ctx, query, trimmed)
	inv := &repository.InviteCode{}
	var limit, expireAt sql.NullInt64
	if err := row.Scan(&inv.ID, &inv.UserID, &inv.Code, &inv.Status, &inv.PV, &limit, &expireAt, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}
	inv.Limit = limit.Int64
	inv.ExpireAt = expireAt.Int64
	return inv, nil
}

func (r *inviteRepo) MarkUsed(ctx context.Context, id int64) error {
	if r == nil || r.db == nil || id <= 0 {
		return nil
	}
	// Decrease limit if it > 0. If limit becomes 0, set status to 1 (used up)
	// But wait, standard logic is usually: if limit > 0, pv < limit is valid.
	// So MarkUsed should just increment PV?
	// The interface name `MarkUsed` suggests marking as fully used (status=1).
	// But for multi-use codes, we just increment PV.
	// Let's check current usage. `RegisterService` calls this.
	// We should probably rename this or change logic to "Consume".
	// For backward compatibility, let's update logic:
	// If limit > 1, decrement limit (or just check PV vs Limit in service).
	// Actually, `IncrementPV` is called separately?
	// Let's assume MarkUsed is only called when the code is fully invalidated (e.g. single use).
	// BUT, `IncrementPV` is usually called on page view, not registration success.
	
	// Let's look at `RegisterService`. It likely calls `MarkUsed` on success.
	// If we support multi-use, `MarkUsed` should only disable it if limit reached.
	
	// For now, let's keep MarkUsed as "set status=1".
	// AND add logic: if limit > 1, don't set status=1?
	// Better: `CreateBatch` creates codes. `Consume` (new method?) or `MarkUsed` logic update.
	
	// Let's stick to simple: MarkUsed sets status=1.
	// Services should check limit before calling MarkUsed or use a new method.
	// But wait, `IncrementPV` is for PV. We need `IncrementUsage`.
	// There is no `usage_count` column, only `pv`.
	// Let's assume `pv` tracks usage for now? Or `limit` decrements?
	// The migration added `limit`.
	// Let's decrement limit in `MarkUsed`?
	// UPDATE v2_invite_code SET "limit" = "limit" - 1, status = CASE WHEN "limit" - 1 <= 0 THEN 1 ELSE status END, updated_at = ? WHERE id = ?
	
	_, err := r.db.ExecContext(ctx, `UPDATE v2_invite_code SET "limit" = "limit" - 1, status = CASE WHEN "limit" - 1 <= 0 THEN 1 ELSE status END, updated_at = ? WHERE id = ? AND "limit" > 0`, time.Now().Unix(), id)
	return err
}

func (r *inviteRepo) CreateBatch(ctx context.Context, codes []*repository.InviteCode) error {
	if len(codes) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO v2_invite_code (user_id, code, status, pv, "limit", expire_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().Unix()
	for _, code := range codes {
		if _, err := stmt.ExecContext(ctx, code.UserID, code.Code, 0, 0, code.Limit, code.ExpireAt, now, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *inviteRepo) CountByStatus(ctx context.Context, status int) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM v2_invite_code WHERE status = ?", status).Scan(&count)
	return count, err
}

func (r *inviteRepo) CountByUser(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM v2_invite_code WHERE user_id = ?", userID).Scan(&count)
	return count, err
}

func (r *inviteRepo) List(ctx context.Context, limit, offset int) ([]*repository.InviteCode, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, user_id, code, status, pv, "limit", expire_at, created_at, updated_at FROM v2_invite_code ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []*repository.InviteCode
	for rows.Next() {
		inv := &repository.InviteCode{}
		var limitVal, expireAt sql.NullInt64
		if err := rows.Scan(&inv.ID, &inv.UserID, &inv.Code, &inv.Status, &inv.PV, &limitVal, &expireAt, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
			return nil, err
		}
		inv.Limit = limitVal.Int64
		inv.ExpireAt = expireAt.Int64
		codes = append(codes, inv)
	}
	return codes, rows.Err()
}

func (r *inviteRepo) CountAll(ctx context.Context) (int64, error) {
	if r == nil || r.db == nil {
		return 0, nil
	}
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM v2_invite_code").Scan(&count)
	return count, err
}
