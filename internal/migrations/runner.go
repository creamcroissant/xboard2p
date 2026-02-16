// 文件路径: internal/migrations/runner.go
// 模块说明: 这是 internal 模块里的 runner 逻辑，下面的注释会用非常通俗的中文帮你理解每一步。
package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func setup() {
	goose.SetDialect("sqlite3")
	goose.SetBaseFS(SQLite)
}

// Up migrates the SQLite schema to the latest version.
func Up(db *sql.DB) error {
	setup()
	return goose.Up(db, "sqlite")
}

// Down rolls back a single migration.
func Down(db *sql.DB) error {
	setup()
	return goose.Down(db, "sqlite")
}

// Status prints migration status.
func Status(db *sql.DB) error {
	setup()
	return goose.Status(db, "sqlite")
}
