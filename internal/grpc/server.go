package grpc

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"github.com/creamcroissant/xboard/internal/grpc/interceptor"
	agentv1 "github.com/creamcroissant/xboard/pkg/pb/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Server 封装 gRPC 服务端。
type Server struct {
	server  *grpc.Server
	logger  *slog.Logger
	address string
}

// Config 保存 gRPC 服务端配置。
type Config struct {
	Address string
	TLS     *TLSConfig
}

// TLSConfig 保存服务端 TLS 配置。
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

// NewServer 创建 gRPC 服务端。
func NewServer(
	cfg Config,
	agentHandler agentv1.AgentServiceServer,
	authInterceptor *interceptor.AuthInterceptor,
	logger *slog.Logger,
) (*Server, error) {
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			interceptor.Recovery(logger),
			interceptor.Logging(logger),
			authInterceptor.Unary(),
		),
		grpc.ChainStreamInterceptor(
			interceptor.StreamRecovery(logger),
			interceptor.StreamLogging(logger),
			authInterceptor.Stream(),
		),
	}

	// TLS 配置
	if cfg.TLS != nil && cfg.TLS.Enabled {
		cert, err := tls.LoadX509KeyPair(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("load TLS certificate: %w", err)
		}
		tlsCfg := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		opts = append(opts, grpc.Creds(credentials.NewTLS(tlsCfg)))
	}

	server := grpc.NewServer(opts...)
	agentv1.RegisterAgentServiceServer(server, agentHandler)

	return &Server{
		server:  server,
		logger:  logger,
		address: cfg.Address,
	}, nil
}

// Start 启动 gRPC 服务。
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", s.address, err)
	}
	return s.Serve(lis)
}

// Serve 在指定 listener 上启动 gRPC 服务。
func (s *Server) Serve(lis net.Listener) error {
	if lis == nil {
		return fmt.Errorf("listener is nil")
	}
	s.logger.Info("gRPC server starting", "address", lis.Addr().String())
	return s.server.Serve(lis)
}

// Handler 返回 gRPC 的 HTTP 处理器。
func (s *Server) Handler() http.Handler {
	return s.server
}

// IsGRPCRequest 判断请求是否为 gRPC。
func IsGRPCRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.ProtoMajor != 2 {
		return false
	}
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	return strings.HasPrefix(contentType, "application/grpc")
}

// Stop 优雅停止 gRPC 服务。
func (s *Server) Stop() {
	s.logger.Info("gRPC server stopping")
	s.server.GracefulStop()
}

// GracefulStop 是 Stop 的别名。
func (s *Server) GracefulStop() {
	s.Stop()
}
