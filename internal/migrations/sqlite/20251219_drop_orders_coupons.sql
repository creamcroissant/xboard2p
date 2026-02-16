-- +goose Up
-- Drop legacy commerce tables now removed from codebase.
DROP INDEX IF EXISTS idx_orders_coupon_user;
DROP INDEX IF EXISTS idx_orders_user_id;
DROP INDEX IF EXISTS idx_orders_trade_no;
DROP TABLE IF EXISTS orders;

DROP INDEX IF EXISTS idx_coupons_code;
DROP TABLE IF EXISTS coupons;

-- +goose Down
-- Recreate legacy tables for rollback compatibility (minimal schema).
CREATE TABLE IF NOT EXISTS orders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trade_no TEXT NOT NULL UNIQUE,
    user_id INTEGER NOT NULL,
    plan_id INTEGER NOT NULL,
    total_amount NUMERIC(20,6) NOT NULL,
    balance_amount NUMERIC(20,6) NOT NULL DEFAULT 0,
    discount_amount NUMERIC(20,6) NOT NULL DEFAULT 0,
    status INTEGER NOT NULL,
    paid_at INTEGER DEFAULT 0,
    coupon TEXT,
    coupon_id INTEGER,
    meta TEXT,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s','now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);
CREATE INDEX IF NOT EXISTS idx_orders_trade_no ON orders(trade_no);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_coupon_user ON orders(coupon_id, user_id);

CREATE TABLE IF NOT EXISTS coupons (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    type INTEGER NOT NULL,
    value INTEGER NOT NULL,
    show INTEGER NOT NULL DEFAULT 0,
    limit_use INTEGER,
    limit_use_with_user INTEGER,
    limit_plan_ids TEXT,
    limit_period TEXT,
    started_at INTEGER NOT NULL,
    ended_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s','now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);
CREATE INDEX IF NOT EXISTS idx_coupons_code ON coupons(code);
