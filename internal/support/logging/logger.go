// 文件路径: internal/support/logging/logger.go
// 模块说明: 这是 internal 模块里的 logger 逻辑，下面的注释会用非常通俗的中文帮你理解每一步。
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Options customize the slog logger construction.
type Options struct {
	Level     slog.Level
	Format    string
	AddSource bool
}

// New returns a slog.Logger configured according to options (JSON by default).
func New(opts Options) *slog.Logger {
	handlerOpts := &slog.HandlerOptions{Level: opts.Level, AddSource: opts.AddSource}

	var handler slog.Handler
	switch strings.ToLower(opts.Format) {
	case "text", "console":
		handler = slog.NewTextHandler(os.Stdout, handlerOpts)
	default:
		handler = slog.NewJSONHandler(os.Stdout, handlerOpts)
	}

	return slog.New(handler)
}
