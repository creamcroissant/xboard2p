-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL,
    balance INTEGER NOT NULL DEFAULT 0,
    plan_id INTEGER DEFAULT 0,
    expired_at INTEGER DEFAULT 0,
    u INTEGER DEFAULT 0,
    d INTEGER DEFAULT 0,
    transfer_enable INTEGER DEFAULT 0,
    commission_balance NUMERIC(20,6) DEFAULT 0,
    status INTEGER DEFAULT 1,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s','now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);

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

CREATE TABLE IF NOT EXISTS payments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT,
    payment TEXT NOT NULL,
    name TEXT NOT NULL,
    icon TEXT,
    config TEXT NOT NULL,
    notify_domain TEXT,
    handling_fee_fixed INTEGER,
    handling_fee_percent NUMERIC(10,4),
    enable INTEGER NOT NULL DEFAULT 0,
    sort INTEGER,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s','now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);

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

CREATE TABLE IF NOT EXISTS server_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    sort INTEGER DEFAULT 0,
    policy TEXT DEFAULT 'manual',
    created_at INTEGER NOT NULL DEFAULT (strftime('%s','now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);

CREATE TABLE IF NOT EXISTS servers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    protocol TEXT NOT NULL,
    config TEXT,
    sort INTEGER DEFAULT 0,
    status INTEGER DEFAULT 1,
    created_at INTEGER NOT NULL DEFAULT (strftime('%s','now')),
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    category TEXT DEFAULT 'general',
    updated_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_orders_trade_no ON orders(trade_no);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_coupon_user ON orders(coupon_id, user_id);
CREATE INDEX IF NOT EXISTS idx_payments_enable ON payments(enable);
CREATE INDEX IF NOT EXISTS idx_coupons_code ON coupons(code);
CREATE INDEX IF NOT EXISTS idx_servers_group_id ON servers(group_id);
CREATE INDEX IF NOT EXISTS idx_settings_category ON settings(category);

INSERT INTO settings(key, value, category)
VALUES
    ('site_name', 'XBoard', 'general'),
    ('default_theme', 'v2board', 'theme')
ON CONFLICT(key) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS servers;
DROP TABLE IF EXISTS server_groups;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS coupons;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS settings;
