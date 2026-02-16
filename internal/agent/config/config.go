package config

import (
	"fmt"
	"os"
	"time"

	
	"gopkg.in/yaml.v3"
)

const (
	defaultProxyPortRangeStart = 30000
	defaultProxyPortRangeEnd   = 40000
	defaultProxyMaxRetries     = 10
	defaultProxyHealthTimeout  = 10 * time.Second
	defaultProxyHealthInterval = 500 * time.Millisecond
	defaultProxyDrainTimeout   = 5 * time.Second
	defaultProxyNftBin         = "/usr/sbin/nft"
	defaultProxyConntrackBin   = "conntrack"
	defaultProxyNftTableName   = "xboard_proxy"
	defaultProxyPIDDir         = "/var/run/xboard/cores"
	defaultProxyCgroupBasePath = "/sys/fs/cgroup/xboard"
)

type Config struct {
	Panel      PanelConfig      `yaml:"panel"`
	Server     ServerConfig     `yaml:"server"`
	GRPC       GRPCConfig       `yaml:"grpc"`
	GRPCServer GRPCServerConfig `yaml:"grpc_server"`
	Interval   IntervalConfig   `yaml:"interval"`
	Core       CoreConfig       `yaml:"core"`
	Traffic    TrafficConfig    `yaml:"traffic"`
	Protocol   ProtocolConfig   `yaml:"protocol"`
	Forwarding ForwardingConfig `yaml:"forwarding"`
	Proxy      ProxyConfig      `yaml:"proxy"`
}

// GRPCConfig holds gRPC client configuration for connecting to Panel
type GRPCConfig struct {
	// Enabled controls whether gRPC transport is used (instead of HTTP)
	Enabled bool `yaml:"enabled"`

	// Address is the Panel gRPC server address (e.g., "panel.example.com:9090")
	Address string `yaml:"address"`

	// TLS configuration
	TLS TLSConfig `yaml:"tls"`

	// Keepalive configuration
	Keepalive KeepaliveConfig `yaml:"keepalive"`

	// Retry configuration
	Retry *RetryConfig `yaml:"retry"`

	// Timeout configuration
	Timeout TimeoutConfig `yaml:"timeout"`
}

// TLSConfig holds TLS settings for gRPC
type TLSConfig struct {
	// Enabled controls whether TLS is used
	Enabled bool `yaml:"enabled"`

	// CertFile is the path to the client certificate (for mutual TLS)
	CertFile string `yaml:"cert_file"`

	// KeyFile is the path to the client key (for mutual TLS)
	KeyFile string `yaml:"key_file"`

	// CAFile is the path to the CA certificate
	CAFile string `yaml:"ca_file"`

	// InsecureSkipVerify skips server certificate verification
	InsecureSkipVerify bool `yaml:"insecure_skip_verify"`
}

// GRPCServerConfig holds gRPC server configuration for Agent.
type GRPCServerConfig struct {
	// Enabled controls whether the Agent gRPC server is started.
	Enabled bool `yaml:"enabled"`

	// Listen address (e.g., ":19090", "127.0.0.1:19090")
	Listen string `yaml:"listen"`

	// AuthToken for gRPC authentication (uses host_token if empty)
	AuthToken string `yaml:"auth_token"`

	// TLS configuration
	TLS ServerTLSConfig `yaml:"tls"`
}

// ServerTLSConfig holds TLS settings for gRPC server.
type ServerTLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// KeepaliveConfig holds keepalive settings for gRPC
type KeepaliveConfig struct {
	// Time is the interval between keepalive pings
	Time time.Duration `yaml:"time"`

	// Timeout is the timeout for keepalive pings
	Timeout time.Duration `yaml:"timeout"`
}

// RetryConfig holds retry settings for gRPC calls.
type RetryConfig struct {
	Enabled         bool          `yaml:"enabled"`
	MaxRetries      int           `yaml:"max_retries"`
	InitialInterval time.Duration `yaml:"initial_interval"`
	MaxInterval     time.Duration `yaml:"max_interval"`
	Multiplier      float64       `yaml:"multiplier"`
}

// TimeoutConfig holds timeout settings for gRPC calls.
type TimeoutConfig struct {
	Default time.Duration `yaml:"default"`
	Connect time.Duration `yaml:"connect"`
}

type ServerConfig struct {
	// Enabled controls whether the HTTP server is started
	Enabled bool `yaml:"enabled"`

	// Listen address (e.g., ":8081", "127.0.0.1:8081")
	Listen string `yaml:"listen"`

	// AuthToken for API authentication (uses host_token if empty)
	AuthToken string `yaml:"auth_token"`
}

type ProtocolConfig struct {
	// ConfigDir is the directory containing sing-box config files
	ConfigDir string `yaml:"config_dir"`

	// SubscribeDir is the directory containing client subscription files
	SubscribeDir string `yaml:"subscribe_dir"`

	// ServiceName is the name of the sing-box service
	ServiceName string `yaml:"service_name"`

	// InitSystem overrides auto-detection: "systemd", "openrc", "runit", "custom"
	InitSystem string `yaml:"init_system"`

	// ValidateCmd is the command to validate config before applying (optional)
	ValidateCmd string `yaml:"validate_cmd"`

	// AutoRestart controls whether to restart service after config change
	AutoRestart bool `yaml:"auto_restart"`

	// PreHook is executed before applying config changes (optional)
	PreHook string `yaml:"pre_hook"`

	// PostHook is executed after applying config changes (optional)
	PostHook string `yaml:"post_hook"`

	// Custom commands for custom init system
	CustomCommands CustomCommands `yaml:"custom_commands"`
}

type CustomCommands struct {
	Start   string `yaml:"start"`
	Stop    string `yaml:"stop"`
	Restart string `yaml:"restart"`
	Reload  string `yaml:"reload"`
	Status  string `yaml:"status"`
	Enable  string `yaml:"enable"`
	Disable string `yaml:"disable"`
}

type PanelConfig struct {
	URL       string `yaml:"url"`        // Panel URL (reserved for future use)
	Token     string `yaml:"token"`      // DEPRECATED: Legacy V2bX node token, no longer supported
	HostToken string `yaml:"host_token"` // Agent Host Token (Required)
	NodeID    int    `yaml:"node_id"`    // DEPRECATED: Legacy V2bX node ID, no longer supported
	NodeType  string `yaml:"node_type"`  // DEPRECATED: Legacy V2bX node type, no longer supported
}

type ForwardingConfig struct {
	// Enabled controls forwarding sync and apply
	Enabled bool `yaml:"enabled"`

	// SyncInterval is the sync interval for forwarding rules
	SyncInterval time.Duration `yaml:"sync_interval"`

	// TableName is the nftables table name
	TableName string `yaml:"table_name"`
}

type ProxyConfig struct {
	Enabled        bool          `yaml:"enabled"`
	PortRangeStart int           `yaml:"port_range_start"`
	PortRangeEnd   int           `yaml:"port_range_end"`
	MaxRetries     int           `yaml:"max_retries"`
	HealthTimeout  time.Duration `yaml:"health_timeout"`
	HealthInterval time.Duration `yaml:"health_interval"`
	DrainTimeout   time.Duration `yaml:"drain_timeout"`
	NftBin         string        `yaml:"nft_bin"`
	ConntrackBin   string        `yaml:"conntrack_bin"`
	NftTableName   string        `yaml:"nft_table_name"`
	PIDDir         string        `yaml:"pid_dir"`
	CgroupBasePath string        `yaml:"cgroup_base_path"`
}


type IntervalConfig struct {
	Sync   int `yaml:"sync"`   // Seconds
	Report int `yaml:"report"` // Seconds
}

type CoreConfig struct {
	TemplatePath string `yaml:"template_path"`
	OutputPath   string `yaml:"output_path"`
	ReloadCmd    string `yaml:"reload_cmd"`
}

type TrafficConfig struct {
	Type      string `yaml:"type"`      // "netio", "none", "dummy", "xray_api"
	Interface string `yaml:"interface"` // Network interface name, e.g., "eth0"; empty for all
	Address   string `yaml:"address"`   // API address for xray_api type, e.g., "127.0.0.1:10085"
}

// Load reads configuration from file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	// Set defaults
	if cfg.Interval.Sync <= 0 {
		cfg.Interval.Sync = 60
	}
	if cfg.Interval.Report <= 0 {
		cfg.Interval.Report = 60
	}

	// Server defaults
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":8081"
	}
	if cfg.Server.AuthToken == "" && cfg.Panel.HostToken != "" {
		cfg.Server.AuthToken = cfg.Panel.HostToken
	}

	// gRPC server defaults
	if cfg.GRPCServer.Listen == "" {
		cfg.GRPCServer.Listen = ":19090"
	}
	if cfg.GRPCServer.AuthToken == "" && cfg.Panel.HostToken != "" {
		cfg.GRPCServer.AuthToken = cfg.Panel.HostToken
	}

	// Protocol defaults
	if cfg.Protocol.ConfigDir == "" {
		cfg.Protocol.ConfigDir = "/etc/sing-box/conf"
	}
	if cfg.Protocol.SubscribeDir == "" {
		cfg.Protocol.SubscribeDir = "/etc/sing-box/subscribe"
	}
	if cfg.Protocol.ServiceName == "" {
		cfg.Protocol.ServiceName = "sing-box"
	}

	// gRPC defaults
	if cfg.GRPC.Keepalive.Time == 0 {
		cfg.GRPC.Keepalive.Time = 30 * time.Second
	}
	if cfg.GRPC.Keepalive.Timeout == 0 {
		cfg.GRPC.Keepalive.Timeout = 10 * time.Second
	}
	if cfg.GRPC.Timeout.Default == 0 {
		cfg.GRPC.Timeout.Default = 10 * time.Second
	}
	if cfg.GRPC.Timeout.Connect == 0 {
		cfg.GRPC.Timeout.Connect = 5 * time.Second
	}
	if cfg.GRPC.Retry == nil {
		cfg.GRPC.Retry = &RetryConfig{
			Enabled:         true,
			MaxRetries:      3,
			InitialInterval: 500 * time.Millisecond,
			MaxInterval:     5 * time.Second,
			Multiplier:      2,
		}
	} else {
		if cfg.GRPC.Retry.MaxRetries == 0 {
			cfg.GRPC.Retry.MaxRetries = 3
		}
		if cfg.GRPC.Retry.InitialInterval == 0 {
			cfg.GRPC.Retry.InitialInterval = 500 * time.Millisecond
		}
		if cfg.GRPC.Retry.MaxInterval == 0 {
			cfg.GRPC.Retry.MaxInterval = 5 * time.Second
		}
		if cfg.GRPC.Retry.Multiplier == 0 {
			cfg.GRPC.Retry.Multiplier = 2
		}
	}

	// Forwarding defaults
	if cfg.Forwarding.SyncInterval == 0 {
		cfg.Forwarding.SyncInterval = 30 * time.Second
	}
	if cfg.Forwarding.TableName == "" {
		cfg.Forwarding.TableName = "xboard_forwarding"
	}

	// Proxy defaults
	if cfg.Proxy.Enabled {
		if cfg.Proxy.PortRangeStart == 0 {
			cfg.Proxy.PortRangeStart = defaultProxyPortRangeStart
		}
		if cfg.Proxy.PortRangeEnd == 0 {
			cfg.Proxy.PortRangeEnd = defaultProxyPortRangeEnd
		}
		if cfg.Proxy.MaxRetries == 0 {
			cfg.Proxy.MaxRetries = defaultProxyMaxRetries
		}
		if cfg.Proxy.HealthTimeout == 0 {
			cfg.Proxy.HealthTimeout = defaultProxyHealthTimeout
		}
		if cfg.Proxy.HealthInterval == 0 {
			cfg.Proxy.HealthInterval = defaultProxyHealthInterval
		}
		if cfg.Proxy.DrainTimeout == 0 {
			cfg.Proxy.DrainTimeout = defaultProxyDrainTimeout
		}
		if cfg.Proxy.NftBin == "" {
			cfg.Proxy.NftBin = defaultProxyNftBin
		}
		if cfg.Proxy.ConntrackBin == "" {
			cfg.Proxy.ConntrackBin = defaultProxyConntrackBin
		}
		if cfg.Proxy.NftTableName == "" {
			cfg.Proxy.NftTableName = defaultProxyNftTableName
		}
		if cfg.Proxy.PIDDir == "" {
			cfg.Proxy.PIDDir = defaultProxyPIDDir
		}
		if cfg.Proxy.CgroupBasePath == "" {
			cfg.Proxy.CgroupBasePath = defaultProxyCgroupBasePath
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate enforces gRPC-only mode and rejects legacy V2bX settings.
func (cfg *Config) Validate() error {
	if cfg.Panel.Token != "" || cfg.Panel.NodeID != 0 || cfg.Panel.NodeType != "" {
		return fmt.Errorf("legacy V2bX/UniProxy mode is no longer supported; migrate to gRPC with grpc.enabled=true, grpc.address, panel.host_token")
	}
	if !cfg.GRPC.Enabled || cfg.GRPC.Address == "" {
		return fmt.Errorf("gRPC transport is required; set grpc.enabled=true and grpc.address")
	}
	if cfg.Panel.HostToken == "" {
		return fmt.Errorf("panel.host_token is required for gRPC authentication")
	}
	if cfg.GRPCServer.Enabled {
		if cfg.GRPCServer.Listen == "" {
			return fmt.Errorf("grpc_server.listen is required when grpc_server is enabled")
		}
		if cfg.GRPCServer.AuthToken == "" {
			return fmt.Errorf("grpc_server.auth_token is required when grpc_server is enabled")
		}
		if cfg.GRPCServer.TLS.Enabled {
			if cfg.GRPCServer.TLS.CertFile == "" || cfg.GRPCServer.TLS.KeyFile == "" {
				return fmt.Errorf("grpc_server.tls.cert_file and grpc_server.tls.key_file are required when grpc_server.tls is enabled")
			}
		}
	}
	if cfg.Proxy.Enabled {
		if cfg.Proxy.PortRangeStart <= 0 || cfg.Proxy.PortRangeEnd <= 0 || cfg.Proxy.PortRangeEnd < cfg.Proxy.PortRangeStart {
			return fmt.Errorf("proxy port range is invalid")
		}
		if cfg.Proxy.MaxRetries <= 0 {
			return fmt.Errorf("proxy max_retries must be positive")
		}
		if cfg.Proxy.HealthTimeout <= 0 || cfg.Proxy.HealthInterval <= 0 || cfg.Proxy.DrainTimeout <= 0 {
			return fmt.Errorf("proxy timeouts must be positive")
		}
	}
	return nil
}
