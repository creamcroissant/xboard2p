package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type LoadOptions struct {
	ConfigPath string
	WorkingDir string
}

func Load() (*Config, error) {
	return LoadWithOptions(LoadOptions{})
}

func LoadWithOptions(opts LoadOptions) (*Config, error) {
	v := viper.New()
	setDefaults(v)
	v.SetConfigType("yaml")
	if err := configureConfigFile(v, opts); err != nil {
		return nil, err
	}
	v.SetEnvPrefix("XBOARD")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := bindEnv(v); err != nil {
		return nil, err
	}
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}
	configDir := configuredDir(v.ConfigFileUsed())
	if err := loadDotEnv(v, configDir); err != nil {
		return nil, err
	}
	if strings.TrimSpace(v.GetString("grpc.addr")) == "" {
		if legacyAddr := strings.TrimSpace(v.GetString("grpc.address")); legacyAddr != "" {
			v.Set("grpc.addr", legacyAddr)
		}
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	resolveRelativePaths(&cfg, configDir)
	return &cfg, nil
}

func configureConfigFile(v *viper.Viper, opts LoadOptions) error {
	if v == nil {
		return fmt.Errorf("viper is nil")
	}
	if configPath := strings.TrimSpace(opts.ConfigPath); configPath != "" {
		absPath, err := filepath.Abs(configPath)
		if err != nil {
			return fmt.Errorf("resolve config path: %w", err)
		}
		v.SetConfigFile(absPath)
		return nil
	}
	v.SetConfigName("config")
	workingDir := strings.TrimSpace(opts.WorkingDir)
	if workingDir != "" {
		v.AddConfigPath(workingDir)
		v.AddConfigPath(filepath.Join(workingDir, "etc"))
	}
	v.AddConfigPath("/etc/xboard/")
	return nil
}

func bindEnv(v *viper.Viper) error {
	bindings := map[string][]string{
		"grpc.enabled":               {"XBOARD_GRPC_ENABLED"},
		"grpc.addr":                  {"XBOARD_GRPC_ADDR"},
		"grpc.tls.enabled":           {"XBOARD_GRPC_TLS_ENABLED"},
		"grpc.tls.cert_file":         {"XBOARD_GRPC_TLS_CERT_FILE"},
		"grpc.tls.key_file":          {"XBOARD_GRPC_TLS_KEY_FILE"},
		"ui.install.enabled":         {"XBOARD_UI_INSTALL_ENABLED", "XBOARD_INSTALL_UI_ENABLED", "INSTALL_UI_ENABLED"},
		"ui.install.dir":             {"XBOARD_UI_INSTALL_DIR", "XBOARD_INSTALL_UI_DIR", "INSTALL_UI_DIR"},
		"scheduler.stat_user_hourly": {"XBOARD_SCHEDULER_STAT_USER_HOURLY"},
		"scheduler.traffic_fetch":    {"XBOARD_SCHEDULER_TRAFFIC_FETCH"},
		"scheduler.email_notify":     {"XBOARD_SCHEDULER_EMAIL_NOTIFY"},
		"scheduler.telegram_notify":  {"XBOARD_SCHEDULER_TELEGRAM_NOTIFY"},
	}
	for key, envs := range bindings {
		args := append([]string{key}, envs...)
		if err := v.BindEnv(args...); err != nil {
			return fmt.Errorf("bind env %s: %w", key, err)
		}
	}
	return nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("http.addr", "0.0.0.0:8080")
	v.SetDefault("http.shutdown_timeout", "15s")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.environment", "production")
	v.SetDefault("database.driver", "sqlite")
	v.SetDefault("database.path", "data/xboard.db")
	v.SetDefault("auth.signing_key", "change-me")
	v.SetDefault("auth.token_ttl", "24h")
	v.SetDefault("auth.issuer", "xboard")
	v.SetDefault("auth.audience", "xboard-client")
	v.SetDefault("auth.leeway", "30s")
	v.SetDefault("auth.bcrypt_cost", 12)
	v.SetDefault("ui.admin.enabled", true)
	v.SetDefault("ui.admin.dir", "web/user-vite/dist")
	v.SetDefault("ui.admin.title", "XBoard Admin")
	v.SetDefault("ui.admin.version", "1.0.0")
	v.SetDefault("ui.admin.hidden_modules", []string{"payment", "ticket", "gift-card", "plugin", "theme"})
	v.SetDefault("ui.user.enabled", true)
	v.SetDefault("ui.user.dir", "web/user-vite/dist")
	v.SetDefault("ui.user.title", "XBoard")
	v.SetDefault("ui.install.enabled", true)
	v.SetDefault("ui.install.dir", "web/install")
	v.SetDefault("scheduler.stat_user_hourly", "@every 5m")
	v.SetDefault("scheduler.traffic_fetch", "@every 1m")
	v.SetDefault("scheduler.email_notify", "@every 1m")
	v.SetDefault("scheduler.telegram_notify", "@every 1m")
}

func loadDotEnv(v *viper.Viper, configDir string) error {
	candidates := []string{}
	if configDir != "" {
		candidates = append(candidates, configDir)
	}
	candidates = append(candidates, "/etc/xboard")
	for _, path := range candidates {
		file := filepath.Clean(filepath.Join(path, ".env"))
		if _, err := os.Stat(file); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("stat .env: %w", err)
		}
		envViper := viper.New()
		envViper.SetConfigFile(file)
		envViper.SetConfigType("env")
		if err := envViper.ReadInConfig(); err != nil {
			return fmt.Errorf("read .env: %w", err)
		}
		bindLegacyEnv(v, envViper)
	}
	return nil
}

func bindLegacyEnv(target *viper.Viper, source *viper.Viper) {
	mappings := map[string]string{
		"HTTP_ADDR":               "http.addr",
		"SHUTDOWN_TIMEOUT":        "http.shutdown_timeout",
		"LOG_LEVEL":               "log.level",
		"LOG_FORMAT":              "log.format",
		"LOG_ADD_SOURCE":          "log.add_source",
		"ENV":                     "log.environment",
		"APP_ENV":                 "log.environment",
		"DB_PATH":                 "database.path",
		"XBOARD_DB_PATH":          "database.path",
		"AUTH_SIGNING_KEY":        "auth.signing_key",
		"XBOARD_AUTH_SIGNING_KEY": "auth.signing_key",
		"APP_KEY":                 "auth.signing_key",
		"AUTH_TOKEN_TTL":          "auth.token_ttl",
		"AUTH_ISSUER":             "auth.issuer",
		"AUTH_AUDIENCE":           "auth.audience",
		"AUTH_LEEWAY":             "auth.leeway",
		"AUTH_BCRYPT_COST":        "auth.bcrypt_cost",
		"ADMIN_UI_ENABLED":        "ui.admin.enabled",
		"ADMIN_UI_DIR":            "ui.admin.dir",
		"ADMIN_UI_TITLE":          "ui.admin.title",
		"ADMIN_UI_VERSION":        "ui.admin.version",
		"ADMIN_UI_BASE_URL":       "ui.admin.base_url",
		"USER_UI_ENABLED":         "ui.user.enabled",
		"USER_UI_DIR":             "ui.user.dir",
		"USER_UI_TITLE":           "ui.user.title",
		"USER_UI_BASE_URL":        "ui.user.base_url",
		"INSTALL_UI_ENABLED":      "ui.install.enabled",
		"INSTALL_UI_DIR":          "ui.install.dir",
	}
	for oldKey, newKey := range mappings {
		if val := source.GetString(oldKey); val != "" {
			target.Set(newKey, val)
		}
	}
}

func configuredDir(configPath string) string {
	if strings.TrimSpace(configPath) == "" {
		return ""
	}
	return filepath.Dir(configPath)
}

func resolveRelativePaths(cfg *Config, baseDir string) {
	if cfg == nil || baseDir == "" {
		return
	}
	cfg.DB.Path = resolveRelativePath(baseDir, cfg.DB.Path)
	cfg.UI.Admin.Dir = resolveRelativePath(baseDir, cfg.UI.Admin.Dir)
	cfg.UI.User.Dir = resolveRelativePath(baseDir, cfg.UI.User.Dir)
	cfg.UI.Install.Dir = resolveRelativePath(baseDir, cfg.UI.Install.Dir)
	cfg.GRPC.TLS.CertFile = resolveRelativePath(baseDir, cfg.GRPC.TLS.CertFile)
	cfg.GRPC.TLS.KeyFile = resolveRelativePath(baseDir, cfg.GRPC.TLS.KeyFile)
}

func resolveRelativePath(baseDir, value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || filepath.IsAbs(trimmed) {
		return trimmed
	}
	return filepath.Clean(filepath.Join(baseDir, trimmed))
}
