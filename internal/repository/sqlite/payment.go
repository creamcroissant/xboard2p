// 文件路径: internal/repository/sqlite/payment.go
// 模块说明: 这是 internal 模块里的 payment 逻辑，下面的注释会用非常通俗的中文帮你理解每一步。
package sqlite

import (
	"context"
	"database/sql"

	"github.com/creamcroissant/xboard/internal/repository"
)

type paymentRepo struct {
	db *sql.DB
}

func (r *paymentRepo) ListEnabled(ctx context.Context) ([]*repository.Payment, error) {
	const query = `SELECT id, uuid, payment, name, icon, config, notify_domain, handling_fee_fixed, handling_fee_percent, enable, sort, created_at, updated_at
        FROM payments WHERE enable = 1 ORDER BY sort ASC, id ASC`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*repository.Payment
	for rows.Next() {
		payment, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

type paymentScanner interface {
	Scan(dest ...any) error
}

func scanPayment(scanner paymentScanner) (*repository.Payment, error) {
	var (
		id                 int64
		uuid               sql.NullString
		paymentCode        sql.NullString
		name               sql.NullString
		icon               sql.NullString
		config             sql.NullString
		notifyDomain       sql.NullString
		handlingFeeFixed   sql.NullInt64
		handlingFeePercent sql.NullFloat64
		enable             sql.NullBool
		sort               sql.NullInt64
		createdAt          int64
		updatedAt          int64
	)

	if err := scanner.Scan(
		&id,
		&uuid,
		&paymentCode,
		&name,
		&icon,
		&config,
		&notifyDomain,
		&handlingFeeFixed,
		&handlingFeePercent,
		&enable,
		&sort,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	payment := &repository.Payment{
		ID:          id,
		UUID:        uuid.String,
		PaymentCode: paymentCode.String,
		Name:        name.String,
		Config:      config.String,
		Enable:      enable.Bool,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
	if icon.Valid {
		payment.Icon = &icon.String
	}
	if notifyDomain.Valid {
		payment.NotifyDomain = &notifyDomain.String
	}
	if handlingFeeFixed.Valid {
		value := handlingFeeFixed.Int64
		payment.HandlingFeeFixed = &value
	}
	if handlingFeePercent.Valid {
		value := handlingFeePercent.Float64
		payment.HandlingFeePercent = &value
	}
	if sort.Valid {
		value := sort.Int64
		payment.Sort = &value
	}
	return payment, nil
}
