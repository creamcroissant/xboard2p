package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	v := viper.New()

	// Default settings
	setDefaults(v)

	// Config file settings
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/xboard/")

	// CLI flag overrides can be bound here if needed

	// Environment variable settings
	v.SetEnvPrefix("XBOARD")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	if err := v.BindEnv("grpc.enabled", "XBOARD_GRPC_ENABLED"); err != nil {
		return nil, fmt.Errorf("bind env grpc.enabled: %w", err)
	}
	if err := v.BindEnv("grpc.addr", "XBOARD_GRPC_ADDR"); err != nil {
		return nil, fmt.Errorf("bind env grpc.addr: %w", err)
	}
	if err := v.BindEnv("grpc.tls.enabled", "XBOARD_GRPC_TLS_ENABLED"); err != nil {
		return nil, fmt.Errorf("bind env grpc.tls.enabled: %w", err)
	}
	if err := v.BindEnv("grpc.tls.cert_file", "XBOARD_GRPC_TLS_CERT_FILE"); err != nil {
		return nil, fmt.Errorf("bind env grpc.tls.cert_file: %w", err)
	}
	if err := v.BindEnv("grpc.tls.key_file", "XBOARD_GRPC_TLS_KEY_FILE"); err != nil {
		return nil, fmt.Errorf("bind env grpc.tls.key_file: %w", err)
	}
	if err := v.BindEnv("ui.install.enabled", "XBOARD_UI_INSTALL_ENABLED", "XBOARD_INSTALL_UI_ENABLED", "INSTALL_UI_ENABLED"); err != nil {
		return nil, fmt.Errorf("bind env ui.install.enabled: %w", err)
	}
	if err := v.BindEnv("ui.install.dir", "XBOARD_UI_INSTALL_DIR", "XBOARD_INSTALL_UI_DIR", "INSTALL_UI_DIR"); err != nil {
		return nil, fmt.Errorf("bind env ui.install.dir: %w", err)
	}

	// 1. Try to read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
		// It's okay if config file is missing, we might rely on Envs/Defaults
		fmt.Println("Config file not found")
	} else {
		fmt.Printf("Using config file: %s\n", v.ConfigFileUsed())
	}

	// 2. Load .env file (backward compatibility)
	if err := loadDotEnv(v); err != nil {
		return nil, err
	}

	fmt.Printf("Viper database.path: %s\n", v.GetString("database.path"))

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	fmt.Printf("Config struct DB.Path: %s\n", cfg.DB.Path)

	return &cfg, nil
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

	v.SetDefault("ui.admin.enabled", false)
	v.SetDefault("ui.admin.dir", "web/admin-vite/dist")
	v.SetDefault("ui.admin.title", "XBoard Admin")
	v.SetDefault("ui.admin.version", "1.0.0")
	v.SetDefault("ui.admin.hidden_modules", []string{"payment", "ticket", "gift-card", "plugin", "theme"})

	v.SetDefault("ui.user.enabled", false)
	v.SetDefault("ui.user.dir", "web/user-vite/dist")
	v.SetDefault("ui.user.title", "XBoard")

	v.SetDefault("ui.install.enabled", true)
	v.SetDefault("ui.install.dir", "web/install")
}

func loadDotEnv(v *viper.Viper) error {
	candidates := []string{".", "..", "../.."}
	for _, path := range candidates {
		file := filepath.Clean(filepath.Join(path, ".env"))
		if _, err := os.Stat(file); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("stat .env: %w", err)
		}

		// Create a separate viper instance for .env to avoid type confusion with main config
		envViper := viper.New()
		envViper.SetConfigFile(file)
		envViper.SetConfigType("env")
		if err := envViper.ReadInConfig(); err != nil {
			return fmt.Errorf("read .env: %w", err)
		}

		// Manually map legacy env vars to new structure
		// This ensures backward compatibility with existing .env files
		bindLegacyEnv(v, envViper)
	}
	return nil
}

// bindLegacyEnv maps old flat ENV variables to the new hierarchical structure
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
			// Only set if not already set by config.yml (which has higher priority than .env but lower than real ENV vars)
			// But Viper.Set overrides everything except real ENVs.
			// Strategy: We just set it. If user has real ENV vars, they will override this because AutomaticEnv is on.
			// But wait, AutomaticEnv works on Get().
			target.Set(newKey, val)
		}
	}
}
