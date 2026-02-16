package service

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/creamcroissant/xboard/internal/agent/access"
	"github.com/creamcroissant/xboard/internal/agent/api"
	"github.com/creamcroissant/xboard/internal/agent/capability"
	"github.com/creamcroissant/xboard/internal/agent/config"
	"github.com/creamcroissant/xboard/internal/agent/core"
	"github.com/creamcroissant/xboard/internal/agent/forwarding"
	agentgrpc "github.com/creamcroissant/xboard/internal/agent/grpc"
	"github.com/creamcroissant/xboard/internal/agent/initsys"
	"github.com/creamcroissant/xboard/internal/agent/monitor"
	"github.com/creamcroissant/xboard/internal/agent/protocol"
	"github.com/creamcroissant/xboard/internal/agent/protocol/subscribe"
	"github.com/creamcroissant/xboard/internal/agent/proxy"
	"github.com/creamcroissant/xboard/internal/agent/server"
	"github.com/creamcroissant/xboard/internal/agent/syncer"
	"github.com/creamcroissant/xboard/internal/agent/traffic"
	"github.com/creamcroissant/xboard/internal/agent/transport"
	agentv1 "github.com/creamcroissant/xboard/pkg/pb/agent/v1"
)

type Agent struct {
	cfg        *config.Config
	grpc       *transport.GRPCClient
	conn       *transport.ConnectionManager
	forward    *forwarding.Manager
	syncer     *syncer.Syncer
	monitor    *monitor.Monitor
	traffic    traffic.Collector
	netio      *traffic.NetIOCollector // Node-level network traffic
	access     *access.Manager         // Access log manager
	protoMgr   *protocol.Manager
	coreMgr    *core.Manager
	switcher   *proxy.Switcher
	server     *server.Server
	grpcServer *agentgrpc.Server
	subParse   *subscribe.Parser    // Subscribe directory parser
	capDet     *capability.Detector // Capability detector

	configETag     string
	usersETag      string
	cachedCaps     *capability.DetectedCapabilities // Cached capabilities
	capsDetectedAt int64                            // Last capability detection time

	// Dynamic intervals
	currentSyncInterval   atomic.Int32
	currentReportInterval atomic.Int32
	updateTickerCh        chan struct{}
}

func New(cfg *config.Config) (*Agent, error) {
	tCollector, err := traffic.NewCollector(cfg.Traffic)
	if err != nil {
		return nil, err
	}

	retryCfg := transport.RetryConfig{}

	// Initialize NetIO collector for node-level traffic
	var netioCollector *traffic.NetIOCollector
	if cfg.Traffic.Type == "netio" {
		netioCollector = traffic.NewNetIOCollector(cfg.Traffic.Interface)
	}

	// Initialize init system
	initSysCfg := initsys.Config{
		Type:        cfg.Protocol.InitSystem,
		ServiceName: cfg.Protocol.ServiceName,
		Custom: initsys.CustomCommands{
			Start:   cfg.Protocol.CustomCommands.Start,
			Stop:    cfg.Protocol.CustomCommands.Stop,
			Restart: cfg.Protocol.CustomCommands.Restart,
			Reload:  cfg.Protocol.CustomCommands.Reload,
			Status:  cfg.Protocol.CustomCommands.Status,
			Enable:  cfg.Protocol.CustomCommands.Enable,
			Disable: cfg.Protocol.CustomCommands.Disable,
		},
	}
	initSys, err := initsys.New(initSysCfg)
	if err != nil {
		return nil, err
	}

	// Initialize protocol manager
	protoCfg := protocol.Config{
		ConfigDir:   cfg.Protocol.ConfigDir,
		ServiceName: cfg.Protocol.ServiceName,
		ValidateCmd: cfg.Protocol.ValidateCmd,
		AutoRestart: cfg.Protocol.AutoRestart,
		PreHook:     cfg.Protocol.PreHook,
		PostHook:    cfg.Protocol.PostHook,
	}
	protoMgr := protocol.NewManager(protoCfg, initSys)

	capDet := capability.NewDetector("", "")
	coreMgr := core.NewManager()
	coreMgr.Register(core.NewSingBoxCore(initSys, capDet, cfg.Protocol.ServiceName, cfg.Protocol.ConfigDir))
	coreMgr.Register(core.NewXrayCore(initSys, capDet, cfg.Protocol.ServiceName, cfg.Traffic.Address, cfg.Protocol.ConfigDir))

	var switcher *proxy.Switcher
	if cfg.Proxy.Enabled {
		switcherOpts := proxy.SwitcherOptions{
			CoreManager: coreMgr,
			OutputPath:  cfg.Core.OutputPath,
			Logger:      slog.Default(),
			Config: proxy.SwitcherConfig{
				PortRangeStart: cfg.Proxy.PortRangeStart,
				PortRangeEnd:   cfg.Proxy.PortRangeEnd,
				MaxRetries:     cfg.Proxy.MaxRetries,
				HealthTimeout:  cfg.Proxy.HealthTimeout,
				HealthInterval: cfg.Proxy.HealthInterval,
				DrainTimeout:   cfg.Proxy.DrainTimeout,
				NftBin:         cfg.Proxy.NftBin,
				ConntrackBin:   cfg.Proxy.ConntrackBin,
				NftTableName:   cfg.Proxy.NftTableName,
				PIDDir:         cfg.Proxy.PIDDir,
				CgroupBasePath: cfg.Proxy.CgroupBasePath,
			},
		}
		created, err := proxy.NewSwitcher(switcherOpts)
		if err != nil {
			return nil, err
		}
		if err := created.Initialize(context.Background()); err != nil {
			return nil, err
		}
		switcher = created
	}

	var srv *server.Server
	if cfg.Server.Enabled {
		srvCfg := server.Config{
			Listen:    cfg.Server.Listen,
			AuthToken: cfg.Server.AuthToken,
		}
		srv = server.NewServer(srvCfg, protoMgr)
	}

	agent := &Agent{
		cfg:      cfg,
		syncer:   syncer.New(cfg.Core),
		monitor:  monitor.New(),
		traffic:  tCollector,
		netio:    netioCollector,
		protoMgr: protoMgr,
		coreMgr:  coreMgr,
		switcher: switcher,
		server:   srv,
		subParse: subscribe.NewParser(cfg.Protocol.SubscribeDir),
		capDet:   capDet, // Use default paths

		updateTickerCh: make(chan struct{}, 1),
	}
	agent.currentSyncInterval.Store(int32(cfg.Interval.Sync))
	agent.currentReportInterval.Store(int32(cfg.Interval.Report))

	if cfg.GRPCServer.Enabled {
		handler := agentgrpc.NewHandler(coreMgr, cfg.Core.OutputPath, slog.Default(), switcher)
		authInterceptor := agentgrpc.NewAuthInterceptor(cfg.GRPCServer.AuthToken)
		grpcCfg := agentgrpc.Config{
			Address: cfg.GRPCServer.Listen,
			TLS: &agentgrpc.TLSConfig{
				Enabled:  cfg.GRPCServer.TLS.Enabled,
				CertFile: cfg.GRPCServer.TLS.CertFile,
				KeyFile:  cfg.GRPCServer.TLS.KeyFile,
			},
		}
		grpcServer, err := agentgrpc.NewServer(grpcCfg, handler, authInterceptor, slog.Default())
		if err != nil {
			return nil, err
		}
		agent.grpcServer = grpcServer
	}
	if cfg.GRPC.Retry != nil {
		retryCfg = transport.RetryConfig{
			Enabled:         cfg.GRPC.Retry.Enabled,
			MaxRetries:      cfg.GRPC.Retry.MaxRetries,
			InitialInterval: cfg.GRPC.Retry.InitialInterval,
			MaxInterval:     cfg.GRPC.Retry.MaxInterval,
			Multiplier:      cfg.GRPC.Retry.Multiplier,
		}
	}
	timeoutCfg := transport.TimeoutConfig{
		Default: cfg.GRPC.Timeout.Default,
		Connect: cfg.GRPC.Timeout.Connect,
	}

	grpcCfg := transport.Config{
		Address: cfg.GRPC.Address,
		Token:   cfg.Panel.HostToken,
		Keepalive: &transport.KeepaliveConfig{
			Time:    cfg.GRPC.Keepalive.Time,
			Timeout: cfg.GRPC.Keepalive.Timeout,
		},
		Retry:   retryCfg,
		Timeout: timeoutCfg,
	}

	if cfg.GRPC.TLS.Enabled {
		grpcCfg.TLS = &transport.TLSConfig{
			Enabled:            true,
			CertFile:           cfg.GRPC.TLS.CertFile,
			KeyFile:            cfg.GRPC.TLS.KeyFile,
			CAFile:             cfg.GRPC.TLS.CAFile,
			InsecureSkipVerify: cfg.GRPC.TLS.InsecureSkipVerify,
		}
	}

	grpcClient, err := transport.NewGRPCClient(grpcCfg)
	if err != nil {
		return nil, err
	}
	agent.grpc = grpcClient
	agent.conn = transport.NewConnectionManager(grpcClient, slog.Default())
	agent.conn.SetOnStateChange(func(state transport.ConnectionState) {
		slog.Info("grpc connection state changed", "state", state.String())
		if state == transport.StateConnected {
			// Trigger immediate sync and report when connected
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
				defer cancel()
				agent.sync(ctx)
				agent.report(ctx)
			}()
		}
	})
	if agent.cfg.Forwarding.Enabled {
		interval := agent.cfg.Forwarding.SyncInterval
		executor := forwarding.NewNFTablesExecutor(agent.cfg.Forwarding.TableName)
		agent.forward = forwarding.NewManager(agent.grpc, executor, interval, slog.Default())
	}

	agent.access = access.NewManager(agent.grpc, agent.coreMgr, slog.Default())

	return agent, nil
}

func (a *Agent) Run(ctx context.Context) {
	// Determine mode
	mode := "agent-host"

	panelAddr := a.cfg.GRPC.Address

	slog.Info("Agent started",
		"mode", mode,
		"transport", "grpc",
		"panel", panelAddr,
		"interval_sync", a.cfg.Interval.Sync,
		"interval_report", a.cfg.Interval.Report,
		"init_system", a.protoMgr.InitSystemType(),
	)

	// Start Agent gRPC server if enabled
	if a.grpcServer != nil {
		go func() {
			if err := a.grpcServer.Start(); err != nil {
				slog.Error("Agent gRPC server error", "error", err)
			}
		}()
		slog.Info("Agent gRPC server enabled", "listen", a.cfg.GRPCServer.Listen)
	}

	// Start HTTP server if enabled
	if a.server != nil {
		go func() {
			if err := a.server.Start(); err != nil {
				slog.Error("HTTP server error", "error", err)
			}
		}()
		slog.Info("Agent HTTP server enabled", "listen", a.cfg.Server.Listen)
	}

	// Start forwarding sync if enabled
	if a.forward != nil {
		go a.forward.Run(ctx)
	}

	// Start access log collector
	if a.access != nil {
		a.access.Start()
	}

	// Initial sync
	a.sync(ctx)

	syncTicker := time.NewTicker(time.Duration(a.currentSyncInterval.Load()) * time.Second)
	reportTicker := time.NewTicker(time.Duration(a.currentReportInterval.Load()) * time.Second)

	defer syncTicker.Stop()
	defer reportTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Agent stopping...")
			if a.server != nil {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				a.server.Shutdown(shutdownCtx)
				cancel()
			}
			if a.grpcServer != nil {
				a.grpcServer.Stop()
			}
			if a.grpc != nil {
				a.grpc.Close()
			}
			if a.access != nil {
				a.access.Stop()
			}
			if a.switcher != nil {
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				_ = a.switcher.Shutdown(shutdownCtx)
				cancel()
			}
			return
		case <-a.updateTickerCh:
			syncInterval := a.currentSyncInterval.Load()
			reportInterval := a.currentReportInterval.Load()
			slog.Info("Updating intervals", "sync", syncInterval, "report", reportInterval)
			syncTicker.Reset(time.Duration(syncInterval) * time.Second)
			reportTicker.Reset(time.Duration(reportInterval) * time.Second)
		case <-syncTicker.C:
			a.sync(ctx)
		case <-reportTicker.C:
			a.report(ctx)
		}
	}
}

func (a *Agent) sync(ctx context.Context) {
	if a.conn != nil {
		state := a.conn.CheckConnection(ctx)
		if state != transport.StateConnected {
			slog.Warn("gRPC connection not ready, skip sync", "state", state.String())
			return
		}
	}
	a.syncGRPC(ctx)
}

func (a *Agent) syncGRPC(ctx context.Context) {
	// NodeID kept for compatibility; gRPC identifies agent host by token
	nodeID := int32(a.cfg.Panel.NodeID)

	// Fetch Config via gRPC
	cfgResp, err := a.grpc.GetConfig(ctx, nodeID, a.configETag)
	if err != nil {
		slog.Error("Failed to fetch config via gRPC", "error", err)
		return
	}

	if !cfgResp.NotModified {
		a.configETag = cfgResp.Etag
		slog.Info("Config updated via gRPC", "version", cfgResp.Version)
		// Apply new config
		if len(cfgResp.ConfigJson) > 0 {
			if err := a.protoMgr.ApplyConfig(ctx, cfgResp.ConfigJson); err != nil {
				slog.Error("Failed to apply config", "error", err)
			} else {
				slog.Info("Successfully applied new config", "version", cfgResp.Version)
			}
		}
	}

	// Fetch Users via gRPC
	usersResp, err := a.grpc.GetUsers(ctx, nodeID, a.usersETag, 0)
	if err != nil {
		slog.Error("Failed to fetch users via gRPC", "error", err)
		return
	}

	if !usersResp.NotModified {
		a.usersETag = usersResp.Etag
		slog.Info("Users updated via gRPC", "count", len(usersResp.Users))

		// Convert users to protocol.UserConfig and inject into config
		if err := a.applyUsers(ctx, usersResp.Users); err != nil {
			slog.Error("Failed to apply users", "error", err)
		} else {
			slog.Info("Successfully applied users to config", "count", len(usersResp.Users))
		}
	}
}

func (a *Agent) report(ctx context.Context) {
	// 1. Collect node-level traffic delta first
	var trafficUpload, trafficDownload uint64
	if a.netio != nil {
		delta, err := a.netio.CollectDelta(ctx)
		if err != nil {
			slog.Error("Failed to collect netio traffic", "error", err)
		} else {
			trafficUpload = delta.Upload
			trafficDownload = delta.Download
		}
	}

	// 2. System Status (with traffic included)
	stat, err := a.monitor.Collect()
	if err != nil {
		slog.Error("Failed to collect system stats", "error", err)
		return
	}

	// Inject traffic into status payload
	stat.TrafficUpload = trafficUpload
	stat.TrafficDownload = trafficDownload

	if a.conn != nil {
		state := a.conn.CheckConnection(ctx)
		if state != transport.StateConnected {
			slog.Warn("gRPC connection not ready, skip report", "state", state.String())
			return
		}
	}
	a.reportGRPC(ctx, stat)

	// 3. User-level Traffic (from traffic collector, e.g., xray_api)
	a.reportUserTraffic(ctx)
}

func (a *Agent) reportGRPC(ctx context.Context, stat api.StatusPayload) {
	// Get capabilities (refresh every hour)
	caps := a.getCapabilities(ctx)

	// Build protobuf status report
	statusReport := &agentv1.StatusReport{
		Timestamp: time.Now().Unix(),
		System: &agentv1.SystemMetrics{
			CpuUsage:        stat.CPU,
			MemoryUsage:     float64(stat.Mem.Used) / float64(stat.Mem.Total) * 100,
			MemoryTotal:     float64(stat.Mem.Total),
			MemoryUsed:      float64(stat.Mem.Used),
			DiskUsage:       float64(stat.Disk.Used) / float64(stat.Disk.Total) * 100,
			DiskTotal:       float64(stat.Disk.Total),
			DiskUsed:        float64(stat.Disk.Used),
			UptimeSeconds:   int64(stat.Uptime),
			ConnectionCount: 0, // TODO: Get connection count if available
			Load1:           stat.Load1,
			Load5:           stat.Load5,
			Load15:          stat.Load15,
			ProcessCount:    int32(stat.ProcessCount),
			TcpCount:        int32(stat.TcpCount),
			UdpCount:        int32(stat.UdpCount),
			// Core capabilities
			CoreVersion:  caps.CoreVersion,
			Capabilities: caps.Capabilities,
			BuildTags:    caps.BuildTags,
		},
		Network: &agentv1.NetworkMetrics{
			UploadBytes:   stat.NetIO.Up,
			DownloadBytes: stat.NetIO.Down,
			UploadDelta:   stat.TrafficUpload,
			DownloadDelta: stat.TrafficDownload,
		},
	}

	// Add core instances
	if a.coreMgr != nil {
		statusReport.Instances = buildCoreInstanceReport(a.coreMgr.ListInstances())
	}

	if configsWithDetails, err := a.protoMgr.ListConfigsWithDetails(); err == nil {
		// Check global service status
		running, _ := a.protoMgr.ServiceStatus(ctx)

		protocols := make([]*agentv1.ProtocolState, 0, len(configsWithDetails))
		for _, cfg := range configsWithDetails {
			state := &agentv1.ProtocolState{
				Name:        cfg.Filename,
				Type:        "sing-box", // Default, will be overridden if detected
				Running:     running,
				ContentHash: cfg.ContentHash,
			}

			// Convert parsed details to protobuf
			if len(cfg.Protocols) > 0 {
				state.Type = cfg.Protocols[0].CoreType // Use detected core type
				details := make([]*agentv1.ProtocolDetails, 0, len(cfg.Protocols))
				for _, p := range cfg.Protocols {
					detail := &agentv1.ProtocolDetails{
						Protocol:   p.Protocol,
						Tag:        p.Tag,
						Listen:     p.Listen,
						Port:       int32(p.Port),
						SourceFile: p.SourceFile,
						CoreType:   p.CoreType,
					}

					// Transport config
					if p.Transport != nil {
						detail.Transport = &agentv1.TransportConfig{
							Type:        p.Transport.Type,
							Path:        p.Transport.Path,
							Host:        p.Transport.Host,
							ServiceName: p.Transport.ServiceName,
						}
					}

					// TLS config
					if p.TLS != nil {
						detail.Tls = &agentv1.TLSConfig{
							Enabled:    p.TLS.Enabled,
							ServerName: p.TLS.ServerName,
							Alpn:       p.TLS.ALPN,
						}
						if p.TLS.Reality != nil {
							detail.Tls.Reality = &agentv1.RealityConfig{
								Enabled:       p.TLS.Reality.Enabled,
								ShortIds:      p.TLS.Reality.ShortIDs,
								ServerName:    p.TLS.Reality.ServerName,
								Fingerprint:   p.TLS.Reality.Fingerprint,
								HandshakeAddr: p.TLS.Reality.HandshakeAddr,
								HandshakePort: int32(p.TLS.Reality.HandshakePort),
								PublicKey:     p.TLS.Reality.PublicKey,
							}
						}
					}

					// Multiplex config
					if p.Multiplex != nil {
						detail.Multiplex = &agentv1.MultiplexConfig{
							Enabled: p.Multiplex.Enabled,
							Padding: p.Multiplex.Padding,
						}
						if p.Multiplex.Brutal != nil {
							detail.Multiplex.Brutal = &agentv1.BrutalConfig{
								Enabled:  p.Multiplex.Brutal.Enabled,
								UpMbps:   int32(p.Multiplex.Brutal.UpMbps),
								DownMbps: int32(p.Multiplex.Brutal.DownMbps),
							}
						}
					}

					// Users
					for _, u := range p.Users {
						detail.Users = append(detail.Users, &agentv1.ProtocolUserInfo{
							Uuid:   u.UUID,
							Flow:   u.Flow,
							Email:  u.Email,
							Method: u.Method,
						})
					}

					details = append(details, detail)
				}
				state.Details = details
			}

			protocols = append(protocols, state)
		}
		statusReport.Protocols = protocols
	} else {
		slog.Error("Failed to list protocol configs", "error", err)
	}

	// Parse subscribe directory for client configs
	if subData, err := a.subParse.Parse(); err == nil && len(subData.Configs) > 0 {
		clientConfigs := make([]*agentv1.ClientConfig, 0, len(subData.Configs))
		for _, cfg := range subData.Configs {
			clientConfig := &agentv1.ClientConfig{
				Name:     cfg.Name,
				Protocol: cfg.Protocol,
				Server:   cfg.Server,
				Port:     int32(cfg.Port),
				// Authentication
				Uuid:     cfg.UUID,
				Password: cfg.Password,
				// Transport
				Network:     cfg.Network,
				Path:        cfg.Path,
				ServiceName: cfg.ServiceName,
				// TLS
				Tls:         cfg.TLS,
				Sni:         cfg.SNI,
				Alpn:        cfg.ALPN,
				Fingerprint: cfg.Fingerprint,
				Insecure:    cfg.Insecure,
				// Reality
				RealityEnabled:   cfg.RealityEnabled,
				RealityPublicKey: cfg.RealityPublicKey,
				RealityShortId:   cfg.RealityShortID,
				// VLESS
				Flow: cfg.Flow,
				// Hysteria2
				HopPorts:    cfg.HopPorts,
				HopInterval: int32(cfg.HopInterval),
				UpMbps:      int32(cfg.UpMbps),
				DownMbps:    int32(cfg.DownMbps),
				// TUIC
				CongestionControl: cfg.CongestionControl,
				// Shadowsocks
				Cipher: cfg.Cipher,
				// ShadowTLS
				ShadowtlsVersion:  int32(cfg.ShadowTLSVersion),
				ShadowtlsPassword: cfg.ShadowTLSPassword,
				// Multiplex
				MuxEnabled: cfg.MuxEnabled,
				MuxPadding: cfg.MuxPadding,
				// Raw configs
				RawConfigs: cfg.RawConfigs,
			}
			clientConfigs = append(clientConfigs, clientConfig)
		}

		statusReport.ClientConfigs = &agentv1.ClientConfigReport{
			Configs:     clientConfigs,
			ContentHash: subData.ContentHash,
		}
		slog.Debug("Parsed subscribe configs", "count", len(clientConfigs))
	} else if err != nil {
		slog.Warn("Failed to parse subscribe directory", "error", err)
	}

	if resp, err := a.grpc.ReportStatus(ctx, statusReport); err != nil {
		slog.Error("Failed to report status via gRPC", "error", err)
	} else {
		slog.Debug("Reported status via gRPC",
			"traffic_up", stat.TrafficUpload,
			"traffic_down", stat.TrafficDownload)

		// Check for interval updates
		updated := false
		if resp.SyncIntervalSeconds > 0 {
			if previous := a.currentSyncInterval.Swap(resp.SyncIntervalSeconds); previous != resp.SyncIntervalSeconds {
				updated = true
			}
		}
		if resp.ReportIntervalSeconds > 0 {
			if previous := a.currentReportInterval.Swap(resp.ReportIntervalSeconds); previous != resp.ReportIntervalSeconds {
				updated = true
			}
		}

		if updated {
			select {
			case a.updateTickerCh <- struct{}{}:
			default:
			}
		}
	}
}

func (a *Agent) reportUserTraffic(ctx context.Context) {
	// Use gRPC for traffic reporting
	samples, err := a.traffic.Collect(ctx)
	if err != nil {
		slog.Error("Failed to collect traffic", "error", err)
		return
	}

	if len(samples) == 0 {
		return
	}

	// Convert to protobuf format
	userTraffic := make([]*agentv1.UserTraffic, len(samples))
	for i, s := range samples {
		userTraffic[i] = &agentv1.UserTraffic{
			UserId:        s.UserID,
			UploadBytes:   s.Upload,
			DownloadBytes: s.Download,
		}
	}

	if _, err := a.grpc.ReportTraffic(ctx, userTraffic); err != nil {
		slog.Error("Failed to push traffic via gRPC", "error", err)
	} else {
		slog.Debug("Pushed traffic samples via gRPC", "count", len(samples))
	}
}

func buildCoreInstanceReport(instances []*core.CoreInstance) []*agentv1.CoreInstance {
	if len(instances) == 0 {
		return nil
	}

	pbInstances := make([]*agentv1.CoreInstance, 0, len(instances))
	for _, inst := range instances {
		pbInstances = append(pbInstances, &agentv1.CoreInstance{
			Id:       inst.ID,
			CoreType: string(inst.CoreType),
			Status:   string(inst.Status),
			ListenPorts: func() []int32 {
				if len(inst.ListenPorts) == 0 {
					return nil
				}
				ports := make([]int32, len(inst.ListenPorts))
				for i, port := range inst.ListenPorts {
					ports[i] = int32(port)
				}
				return ports
			}(),
			ConfigPath: inst.ConfigPath,
			ConfigHash: inst.ConfigHash,
			Pid:        int32(inst.PID),
			StartedAt:  inst.StartedAt,
			Error:      inst.Error,
		})
	}
	return pbInstances
}

// getCapabilities returns cached or fresh capabilities
// Capabilities are cached for 1 hour to avoid excessive command executions
func (a *Agent) getCapabilities(ctx context.Context) *capability.DetectedCapabilities {
	now := time.Now().Unix()
	cacheExpiry := int64(3600) // 1 hour

	// Return cached if still valid
	if a.cachedCaps != nil && now-a.capsDetectedAt < cacheExpiry {
		return a.cachedCaps
	}

	// Detect fresh capabilities
	caps, err := a.capDet.Detect(ctx)
	if err != nil {
		slog.Warn("Failed to detect capabilities", "error", err)
		// Return empty capabilities on error
		return &capability.DetectedCapabilities{
			CoreType:     "unknown",
			Capabilities: []string{},
			BuildTags:    []string{},
		}
	}

	// Cache the result
	a.cachedCaps = caps
	a.capsDetectedAt = now

	slog.Info("Detected core capabilities",
		"core_type", caps.CoreType,
		"version", caps.CoreVersion,
		"capabilities", caps.Capabilities,
		"build_tags", caps.BuildTags)

	return caps
}

// applyUsers converts gRPC UserInfo to protocol.UserConfig and injects them into the config.
func (a *Agent) applyUsers(ctx context.Context, users []*agentv1.UserInfo) error {
	if len(users) == 0 {
		return nil
	}

	// Convert gRPC UserInfo to protocol.UserConfig
	userConfigs := make([]protocol.UserConfig, 0, len(users))
	for _, u := range users {
		userConfigs = append(userConfigs, protocol.UserConfig{
			UUID:    u.Uuid,
			Email:   u.Email,
			Enabled: u.Enabled,
		})
	}

	// Detect core type and use appropriate injection method
	coreType := a.protoMgr.DetectCoreType()
	slog.Debug("Detected core type for user injection", "core_type", coreType)

	switch coreType {
	case "xray":
		return a.protoMgr.InjectUsersXray(ctx, userConfigs)
	case "sing-box":
		return a.protoMgr.InjectUsers(ctx, userConfigs)
	default:
		// Try Sing-box first as it's the default
		if err := a.protoMgr.InjectUsers(ctx, userConfigs); err != nil {
			// If that fails, try Xray
			return a.protoMgr.InjectUsersXray(ctx, userConfigs)
		}
		return nil
	}
}
