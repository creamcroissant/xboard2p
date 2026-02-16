package transport

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"

	agentv1 "github.com/creamcroissant/xboard/pkg/pb/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// GRPCClient wraps a gRPC connection to the Panel
type GRPCClient struct {
	conn   *grpc.ClientConn
	client agentv1.AgentServiceClient
	token  string
	config Config

	connManager *ConnectionManager

	mu        sync.RWMutex
	connected bool
}

// Config holds gRPC client configuration
type Config struct {
	Address   string
	Token     string
	TLS       *TLSConfig
	Keepalive *KeepaliveConfig
	Retry     RetryConfig
	Timeout   TimeoutConfig
}

// TimeoutConfig holds timeout settings for gRPC calls.
type TimeoutConfig struct {
	Default time.Duration
	Connect time.Duration
}

// TLSConfig holds TLS settings
type TLSConfig struct {
	Enabled            bool
	CertFile           string
	KeyFile            string
	CAFile             string
	InsecureSkipVerify bool
}

// KeepaliveConfig holds keepalive settings
type KeepaliveConfig struct {
	Time    time.Duration
	Timeout time.Duration
}

// NewGRPCClient creates a new gRPC client to the Panel
func NewGRPCClient(cfg Config) (*GRPCClient, error) {
	// Set defaults
	if cfg.Keepalive == nil {
		cfg.Keepalive = &KeepaliveConfig{
			Time:    30 * time.Second,
			Timeout: 10 * time.Second,
		}
	}

	if cfg.Timeout.Default == 0 {
		cfg.Timeout.Default = 10 * time.Second
	}
	if cfg.Timeout.Connect == 0 {
		cfg.Timeout.Connect = 5 * time.Second
	}

	cfg.Retry = normalizeRetryConfig(cfg.Retry)

	opts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.Keepalive.Time,
			Timeout:             cfg.Keepalive.Timeout,
			PermitWithoutStream: true,
		}),
	}

	// TLS configuration
	if cfg.TLS != nil && cfg.TLS.Enabled {
		tlsCfg, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, fmt.Errorf("build TLS config: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	dialCtx := context.Background()
	var cancel context.CancelFunc
	if cfg.Timeout.Connect > 0 {
		dialCtx, cancel = context.WithTimeout(dialCtx, cfg.Timeout.Connect)
		defer cancel()
	}

	conn, err := grpc.DialContext(dialCtx, cfg.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gRPC client: %w", err)
	}

	client := &GRPCClient{
		conn:   conn,
		client: agentv1.NewAgentServiceClient(conn),
		token:  cfg.Token,
		config: cfg,
	}
	client.connManager = NewConnectionManager(client, nil)
	return client, nil
}

func buildTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	// Load CA certificate if provided
	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsCfg.RootCAs = caCertPool
	}

	// Load client certificate if provided (for mutual TLS)
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load client certificate: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return tlsCfg, nil
}

// withAuth adds authentication metadata to the context
func (c *GRPCClient) withAuth(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+c.token)
}

// Close closes the gRPC connection
func (c *GRPCClient) Close() error {
	return c.conn.Close()
}

// IsConnected returns true if the connection is ready
func (c *GRPCClient) IsConnected() bool {
	return c.conn.GetState() == connectivity.Ready
}

// WaitForReady waits for the connection to be ready
func (c *GRPCClient) WaitForReady(ctx context.Context) error {
	for {
		state := c.conn.GetState()
		if state == connectivity.Ready {
			return nil
		}
		if !c.conn.WaitForStateChange(ctx, state) {
			return ctx.Err()
		}
	}
}

// IsHealthy returns true if the connection is ready.
func (c *GRPCClient) IsHealthy() bool {
	if c.connManager == nil {
		return c.IsConnected()
	}
	return c.connManager.IsHealthy()
}

// SetConnectionStateHook sets a callback for connection state changes.
func (c *GRPCClient) SetConnectionStateHook(fn func(ConnectionState)) {
	if c.connManager != nil {
		c.connManager.SetOnStateChange(fn)
	}
}

func (c *GRPCClient) Heartbeat(ctx context.Context) (*agentv1.HeartbeatResponse, error) {
	return callUnary(ctx, c, CallConfig{}, func(ctx context.Context) (*agentv1.HeartbeatResponse, error) {
		return c.client.Heartbeat(ctx, &agentv1.HeartbeatRequest{
			Timestamp: time.Now().Unix(),
		})
	})
}

// ReportStatus reports system metrics and network traffic
func (c *GRPCClient) ReportStatus(ctx context.Context, status *agentv1.StatusReport) (*agentv1.StatusResponse, error) {
	return callUnary(ctx, c, CallConfig{}, func(ctx context.Context) (*agentv1.StatusResponse, error) {
		return c.client.ReportStatus(ctx, status)
	})
}

// CallConfig controls timeout and retry behavior for an RPC.
type CallConfig struct {
	Timeout   time.Duration
	Retry     RetryConfig
	SkipRetry bool
}

func (c *GRPCClient) call(ctx context.Context, cfg CallConfig, fn func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if c.connManager != nil {
		c.connManager.CheckConnection(ctx)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = c.config.Timeout.Default
	}

	callCtx := ctx
	var cancel context.CancelFunc
	if timeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	retryCfg := cfg.Retry
	if retryCfg.MaxRetries == 0 {
		retryCfg = c.config.Retry
	}
	if cfg.SkipRetry {
		retryCfg.Enabled = false
	}

	err := DoWithRetry(callCtx, retryCfg, func(attemptCtx context.Context) error {
		attemptCtx = c.withAuth(attemptCtx)
		return fn(attemptCtx)
	})
	if err != nil {
		if c.connManager != nil {
			c.connManager.RecordError(err)
		}
		return err
	}

	if c.connManager != nil {
		c.connManager.RecordSuccess()
	}
	return nil
}

func callUnary[T any](ctx context.Context, c *GRPCClient, cfg CallConfig, fn func(context.Context) (*T, error)) (*T, error) {
	var resp *T
	err := c.call(ctx, cfg, func(inner context.Context) error {
		var err error
		resp, err = fn(inner)
		return err
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GetConfig fetches node configuration
func (c *GRPCClient) GetConfig(ctx context.Context, nodeID int32, etag string) (*agentv1.ConfigResponse, error) {
	return callUnary(ctx, c, CallConfig{}, func(ctx context.Context) (*agentv1.ConfigResponse, error) {
		return c.client.GetConfig(ctx, &agentv1.ConfigRequest{
			NodeId: nodeID,
			Etag:   etag,
		})
	})
}

// GetUsers fetches user list for the node
func (c *GRPCClient) GetUsers(ctx context.Context, nodeID int32, etag string, sinceVersion int64) (*agentv1.UsersResponse, error) {
	return callUnary(ctx, c, CallConfig{}, func(ctx context.Context) (*agentv1.UsersResponse, error) {
		return c.client.GetUsers(ctx, &agentv1.UsersRequest{
			NodeId:       nodeID,
			Etag:         etag,
			SinceVersion: sinceVersion,
		})
	})
}

// ReportTraffic reports user-level traffic data
func (c *GRPCClient) ReportTraffic(ctx context.Context, traffic []*agentv1.UserTraffic) (*agentv1.TrafficResponse, error) {
	return callUnary(ctx, c, CallConfig{}, func(ctx context.Context) (*agentv1.TrafficResponse, error) {
		return c.client.ReportTraffic(ctx, &agentv1.TrafficReport{
			Timestamp:   time.Now().Unix(),
			UserTraffic: traffic,
		})
	})
}

// ReportAlive reports active user IDs
func (c *GRPCClient) ReportAlive(ctx context.Context, userIDs []int64) (*agentv1.AliveResponse, error) {
	return callUnary(ctx, c, CallConfig{}, func(ctx context.Context) (*agentv1.AliveResponse, error) {
		return c.client.ReportAlive(ctx, &agentv1.AliveReport{
			Timestamp: time.Now().Unix(),
			UserIds:   userIDs,
		})
	})
}

// StatusStream opens a bidirectional stream for real-time status updates
func (c *GRPCClient) StatusStream(ctx context.Context) (agentv1.AgentService_StatusStreamClient, error) {
	ctx = c.withAuth(ctx)
	return c.client.StatusStream(ctx)
}

// ReportForwardingStatus reports forwarding apply status.
func (c *GRPCClient) ReportForwardingStatus(ctx context.Context, report *agentv1.ForwardingStatusReport) (*agentv1.StatusResponse, error) {
	return callUnary(ctx, c, CallConfig{}, func(ctx context.Context) (*agentv1.StatusResponse, error) {
		return c.client.ReportForwardingStatus(ctx, report)
	})
}

// GetForwardingRules fetches forwarding rules for agent host.
func (c *GRPCClient) GetForwardingRules(ctx context.Context, version int64) (*agentv1.ForwardingRulesResponse, error) {
	return callUnary(ctx, c, CallConfig{}, func(ctx context.Context) (*agentv1.ForwardingRulesResponse, error) {
		return c.client.GetForwardingRules(ctx, &agentv1.ForwardingRulesRequest{Version: version})
	})
}

// ReportAccessLogs reports access logs
func (c *GRPCClient) ReportAccessLogs(ctx context.Context, report *agentv1.AccessLogReport) (*agentv1.AccessLogResponse, error) {
	return callUnary(ctx, c, CallConfig{}, func(ctx context.Context) (*agentv1.AccessLogResponse, error) {
		return c.client.ReportAccessLogs(ctx, report)
	})
}

// Client returns the underlying AgentServiceClient for advanced usage
func (c *GRPCClient) Client() agentv1.AgentServiceClient {
	return c.client
}
