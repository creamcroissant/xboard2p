package template

// UnifiedInbound 是与核心无关的入站模型，供转换器使用。
type UnifiedInbound struct {
	Tag       string            `json:"tag"`
	Protocol  string            `json:"protocol"` // vless, vmess, shadowsocks, trojan
	Listen    string            `json:"listen"`
	Port      int               `json:"port"`
	Transport *UnifiedTransport `json:"transport,omitempty"`
	TLS       *UnifiedTLS       `json:"tls,omitempty"`
	Users     []UnifiedUser     `json:"users,omitempty"`
	Options   map[string]any    `json:"options,omitempty"`
}

// UnifiedTransport 描述跨核心的传输配置。
type UnifiedTransport struct {
	Type        string            `json:"type"` // tcp, ws, grpc, http, h2, quic
	Path        string            `json:"path,omitempty"`
	Host        string            `json:"host,omitempty"`
	ServiceName string            `json:"service_name,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// UnifiedTLS 描述跨核心的 TLS 配置。
type UnifiedTLS struct {
	Enabled    bool            `json:"enabled"`
	ServerName string          `json:"server_name,omitempty"`
	ALPN       []string        `json:"alpn,omitempty"`
	CertPath   string          `json:"cert_path,omitempty"`
	KeyPath    string          `json:"key_path,omitempty"`
	Reality    *UnifiedReality `json:"reality,omitempty"`
}

// UnifiedReality 描述跨核心的 Reality 配置。
type UnifiedReality struct {
	Enabled         bool     `json:"enabled"`
	PrivateKey      string   `json:"private_key,omitempty"`
	PublicKey       string   `json:"public_key,omitempty"`
	ShortIDs        []string `json:"short_ids,omitempty"`
	ServerNames     []string `json:"server_names,omitempty"`       // 支持多 SNI 防探测
	HandshakeServer string   `json:"handshake_server,omitempty"`
	HandshakePort   int      `json:"handshake_port,omitempty"`
	Fingerprint     string   `json:"fingerprint,omitempty"`
}

// UnifiedUser 描述跨核心的用户配置。
type UnifiedUser struct {
	UUID     string `json:"uuid,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	Flow     string `json:"flow,omitempty"`
	Method   string `json:"method,omitempty"`
}
