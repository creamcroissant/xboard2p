package config

import (
	"log/slog"
	"time"
)

// Config 汇总应用的全部配置。
type Config struct {
	HTTP     HTTPConfig     `mapstructure:"http"`
	GRPC     GRPCConfig     `mapstructure:"grpc"`
	Log      LogConfig      `mapstructure:"log"`
	DB       DBConfig       `mapstructure:"database"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Security SecurityConfig `mapstructure:"security"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	UI       UIConfig       `mapstructure:"ui"`
	Cores    []CoreConfig   `mapstructure:"cores"`
	Nodes    []NodeConfig   `mapstructure:"nodes"`
}

// GRPCConfig 定义 Agent 通信所需的 gRPC 服务配置。
type GRPCConfig struct {
	Enabled bool           `mapstructure:"enabled"`
	Addr    string         `mapstructure:"addr"`
	TLS     GRPCTLSConfig  `mapstructure:"tls"`
}

// GRPCTLSConfig 定义 gRPC 服务的 TLS 配置。
type GRPCTLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// SecurityConfig 定义安全相关配置。
type SecurityConfig struct {
	SubscribeObfuscation bool `mapstructure:"subscribe_obfuscation"`
}

// MetricsConfig 定义 Prometheus 指标配置。
type MetricsConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	Namespace string   `mapstructure:"namespace"`
	Subsystem string   `mapstructure:"subsystem"`
	Token     string   `mapstructure:"token"`
	Buckets   []float64 `mapstructure:"buckets"`
}

// HTTPConfig 定义 HTTP 服务配置。
type HTTPConfig struct {
	Addr            string        `mapstructure:"addr"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// LogConfig 定义日志配置。
type LogConfig struct {
	Level       string `mapstructure:"level"`
	Format      string `mapstructure:"format"`
	AddSource   bool   `mapstructure:"add_source"`
	Environment string `mapstructure:"environment"`
}

// DBConfig 定义数据库配置。
type DBConfig struct {
	Driver string `mapstructure:"driver"`
	Path   string `mapstructure:"path"`
}

// AuthConfig 定义认证配置。
type AuthConfig struct {
	SigningKey string        `mapstructure:"signing_key"`
	TokenTTL   time.Duration `mapstructure:"token_ttl"`
	Issuer     string        `mapstructure:"issuer"`
	Audience   string        `mapstructure:"audience"`
	Leeway     time.Duration `mapstructure:"leeway"`
	BcryptCost int           `mapstructure:"bcrypt_cost"`
}

// UIConfig 定义前端静态资源配置。
type UIConfig struct {
	Admin   AdminUIConfig   `mapstructure:"admin"`
	User    UserUIConfig    `mapstructure:"user"`
	Install InstallUIConfig `mapstructure:"install"`
}

type AdminUIConfig struct {
	Enabled       bool     `mapstructure:"enabled"`
	Dir           string   `mapstructure:"dir"`
	BaseURL       string   `mapstructure:"base_url"`
	Title         string   `mapstructure:"title"`
	Version       string   `mapstructure:"version"`
	HiddenModules []string `mapstructure:"hidden_modules"`
}

type UserUIConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Dir     string `mapstructure:"dir"`
	BaseURL string `mapstructure:"base_url"`
	Title   string `mapstructure:"title"`
}

type InstallUIConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Dir     string `mapstructure:"dir"`
}

// CoreConfig 定义代理核心配置（Xray/Sing-box）。
type CoreConfig struct {
	Type         string         `mapstructure:"type"`
	Name         string         `mapstructure:"name"`
	OriginalPath string         `mapstructure:"original_path"`
	Log          CoreLogConfig  `mapstructure:"log"`
}

type CoreLogConfig struct {
	Level     string `mapstructure:"level"`
	Timestamp bool   `mapstructure:"timestamp"`
}

// NodeConfig 定义节点实例配置。
type NodeConfig struct {
	Core                   string     `mapstructure:"core"`
	NodeID                 int        `mapstructure:"node_id"`
	NodeType               string     `mapstructure:"node_type"`
	ApiHost                string     `mapstructure:"api_host"`
	ApiKey                 string     `mapstructure:"api_key"`
	Timeout                int        `mapstructure:"timeout"`
	ListenIP               string     `mapstructure:"listen_ip"`
	SendIP                 string     `mapstructure:"send_ip"`
	DeviceOnlineMinTraffic int64      `mapstructure:"device_online_min_traffic"`
	CertConfig             CertConfig `mapstructure:"cert_config"`
}

type CertConfig struct {
	CertMode   string `mapstructure:"cert_mode"`
	CertDomain string `mapstructure:"cert_domain"`
	Provider   string `mapstructure:"provider"`
}

func (c LogConfig) SlogLevel() slog.Level {
	switch c.Level {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}