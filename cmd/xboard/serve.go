package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/creamcroissant/xboard/internal/api"
	"github.com/creamcroissant/xboard/internal/async"
	"github.com/creamcroissant/xboard/internal/bootstrap"
	"github.com/creamcroissant/xboard/internal/config"
	internalgrpc "github.com/creamcroissant/xboard/internal/grpc"
	"github.com/creamcroissant/xboard/internal/grpc/handler"
	"github.com/creamcroissant/xboard/internal/grpc/interceptor"
	"github.com/creamcroissant/xboard/internal/job"
	"github.com/creamcroissant/xboard/internal/migrations"
	"github.com/creamcroissant/xboard/internal/protocol"
	"github.com/creamcroissant/xboard/internal/repository/sqlite"
	"github.com/creamcroissant/xboard/internal/service"
	"github.com/creamcroissant/xboard/internal/support/i18n"
	"github.com/creamcroissant/xboard/internal/support/logging"
	"github.com/creamcroissant/xboard/internal/template"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the XBoard server",
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	bootTime := time.Now().UTC()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(logging.Options{
		Level:     cfg.Log.SlogLevel(),
		Format:    cfg.Log.Format,
		AddSource: cfg.Log.AddSource,
	})

	db, err := bootstrap.OpenSQLite(cfg.DB.Path)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := migrations.Up(db); err != nil {
		return err
	}

	resolvedSigningKey, signingKeySource, err := bootstrap.ResolveJWTSigningKey(ctx, db, cfg.Auth.SigningKey, time.Now)
	if err != nil {
		return err
	}
	cfg.Auth.SigningKey = resolvedSigningKey

	switch signingKeySource {
	case bootstrap.JWTSigningKeySourceConfig:
		logger.Info("jwt signing key loaded", "source", "config")
	case bootstrap.JWTSigningKeySourceSettings:
		logger.Info("jwt signing key loaded", "source", "settings")
	case bootstrap.JWTSigningKeySourceGenerated:
		logger.Info("jwt signing key generated", "source", "generated-and-persisted")
	default:
		logger.Info("jwt signing key loaded", "source", "unknown")
	}

	// Migrate legacy config to bootstrap config structure for compatibility
	// TODO: Fully replace bootstrap.Config with config.Config throughout the app
	legacyCfg := &bootstrap.Config{
		HTTP: bootstrap.HTTPConfig{
			Addr:            cfg.HTTP.Addr,
			ShutdownTimeout: cfg.HTTP.ShutdownTimeout,
		},
		Log: bootstrap.LogConfig{
			Level:       cfg.Log.SlogLevel(),
			Format:      cfg.Log.Format,
			AddSource:   cfg.Log.AddSource,
			Environment: cfg.Log.Environment,
		},
		DB: bootstrap.DBConfig{
			SQLitePath: cfg.DB.Path,
		},
		Auth: bootstrap.AuthConfig{
			SigningKey: cfg.Auth.SigningKey,
			TokenTTL:   cfg.Auth.TokenTTL,
			Issuer:     cfg.Auth.Issuer,
			Audience:   cfg.Auth.Audience,
			Leeway:     cfg.Auth.Leeway,
			BcryptCost: cfg.Auth.BcryptCost,
		},
		UI: bootstrap.UIConfig{
			Admin: bootstrap.AdminUIConfig{
				Enabled:       cfg.UI.Admin.Enabled,
				Dir:           cfg.UI.Admin.Dir,
				Title:         cfg.UI.Admin.Title,
				Version:       cfg.UI.Admin.Version,
				BaseURL:       cfg.UI.Admin.BaseURL,
				HiddenModules: cfg.UI.Admin.HiddenModules,
			},
			Install: bootstrap.InstallUIConfig{
				Enabled: cfg.UI.Install.Enabled,
				Dir:     cfg.UI.Install.Dir,
			},
		},
	}

	infra, err := bootstrap.BuildInfrastructure(legacyCfg, logger)
	if err != nil {
		return err
	}

	store := sqlite.NewStore(db)

	// Services initialization
	inviteService := service.NewInviteService(store.InviteCodes(), store.Users())
	captchaService := service.NewCaptchaService(store.Settings(), nil)
	notificationQueue := async.NewNotificationQueue()
	queuedNotifier := async.NewQueueNotifier(notificationQueue)
	verifyService := service.NewVerificationService(infra.Cache, queuedNotifier, store.Settings(), store.Users(), captchaService)
	passwordService := service.NewPasswordService(store.Users(), infra.Hasher, verifyService, infra.Cache)
	registrationService := service.NewRegistrationService(store.Users(), inviteService, store.Settings(), infra.Hasher, verifyService, infra.Cache)
	mailLinkService := service.NewMailLinkService(store.Users(), store.Settings(), queuedNotifier, infra.Cache)
	commService := service.NewCommService(store.Settings(), store.Plugins())
	planService := service.NewPlanService(store.Plans(), store.Users(), store.Settings(), store.ServerGroups())
	i18nManager, err := i18n.NewManager(
		i18n.WithLogger(logger),
		i18n.WithDefaultLang("en-US"),
	)
	if err != nil {
		return err
	}

	adminPlanService := service.NewAdminPlanService(store.Plans(), i18nManager)
	serverTelemetryService := service.NewServerTelemetryServiceWithLogger(infra.Cache, store.Settings(), store.Servers(), store.StatServers(), logger)
	adminUserService := service.NewAdminUserService(
		store.Users(),
		store.Plans(),
		store.ServerGroups(),
		store.Settings(),
		serverTelemetryService,
		infra.Hasher,
		i18nManager,
	)
	adminServerService := service.NewAdminServerService(store.ServerGroups(), store.ServerRoutes(), store.Servers(), i18nManager)
	adminStatService := service.NewAdminStatService(store.StatUsers(), store.Users())
	adminNodeStatService := service.NewAdminNodeStatService(store.StatServers())
	adminNoticeService := service.NewAdminNoticeService(store.Notices(), i18nManager)
	adminKnowledgeService := service.NewAdminKnowledgeService(store.Knowledge(), i18nManager)
	userKnowledgeService := service.NewUserKnowledgeService(store.Knowledge(), store.Users(), store.Settings())
	userNoticeService := service.NewUserNoticeService(store.Notices(), store.UserNoticeReads())
	userStatService := service.NewUserStatService(store.StatUsers())
	protocolManager := protocol.NewManager(
		protocol.NewGeneralBuilder(),
		protocol.NewClashBuilder(),
		protocol.NewSurgeBuilder(),
		protocol.NewSingboxBuilder(),
	)
	serverAuthService := service.NewServerAuthService(store.Settings(), store.Servers())
	serverNodeService := service.NewServerNodeService(store.Users(), store.ServerRoutes(), store.Settings())

	// Multi-accumulator for multi-granularity statistics (hourly, daily, monthly)
	multiAccumulator := job.NewMultiAccumulator(3) // 0=hourly, 1=daily, 2=monthly
	serverTrafficService := service.NewServerTrafficService(store.Users(), multiAccumulator)
	userTrafficService := service.NewUserTrafficServiceWithCollector(store.UserTraffic(), store.Users(), multiAccumulator, notificationQueue, store.Settings())
	userServerSelectionService := service.NewUserServerSelectionService(store.UserTraffic())
	trafficQueue := async.NewTrafficQueue()
	subLogQueue := async.NewSubscriptionLogQueue(store.SubscriptionLogs(), logger)
	installService := service.NewInstallService(store.Users(), infra.Hasher, i18nManager)

	adminSystemService := service.NewAdminSystemService(service.AdminSystemOptions{
		Version:           cfg.UI.Admin.Version,
		Environment:       cfg.Log.Environment,
		StartedAt:         bootTime,
		NotificationQueue: notificationQueue,
		TrafficQueue:      trafficQueue,
		Users:             store.Users(),
		Servers:           store.Servers(),
		AgentHosts:        store.AgentHosts(),
		I18n:              i18nManager,
	})
	adminSystemSettingsService := service.NewAdminSystemSettingsService(service.AdminSystemSettingsOptions{
		Settings:          store.Settings(),
		NotificationQueue: notificationQueue,
		Audit:             infra.Audit,
	})

	agentHostService := service.NewAgentHostService(store.AgentHosts(), store.Servers(), store.ServerClientConfigs(), store.ConfigTemplates(), store.Users())
	agentService := service.NewAgentService(store.Servers(), store.Users())
	forwardingService := service.NewForwardingServiceWithLogger(store.ForwardingRules(), store.ForwardingRuleLogs(), store.AgentHosts(), logger)
	converterRegistry := template.NewConverterRegistry(&template.SingBoxConverter{}, &template.XrayConverter{})
	agentCoreService := service.NewAgentCoreService(
		store.AgentHosts(),
		store.AgentCoreInstances(),
		store.AgentCoreSwitchLogs(),
		store.ConfigTemplates(),
		converterRegistry,
		logger,
	)
	accessLogService := service.NewAccessLogService(store)
	shortLinkService := service.NewShortLinkService(store.ShortLinks(), store.Users(), store.Settings())

	scheduler := job.NewScheduler(logger)

	// Multi-granularity stat user jobs for traffic aggregation
	// Each job uses its own accumulator from the multi-accumulator
	// Hourly: runs every 5 minutes (modified to 10s for testing), aggregates to hourly buckets
	statUserJobHourly := job.NewStatUserJobWithType(multiAccumulator.Get(job.RecordTypeHourly), store.StatUsers(), logger, job.RecordTypeHourly)
	if _, err := scheduler.Register("@every 10s", statUserJobHourly); err != nil {
		return err
	}
	// Daily: runs every hour, aggregates to daily buckets
	statUserJobDaily := job.NewStatUserJobWithType(multiAccumulator.Get(job.RecordTypeDaily), store.StatUsers(), logger, job.RecordTypeDaily)
	if _, err := scheduler.Register("@every 1h", statUserJobDaily); err != nil {
		return err
	}
	// Monthly: runs at 00:05 every day, aggregates to monthly buckets
	statUserJobMonthly := job.NewStatUserJobWithType(multiAccumulator.Get(job.RecordTypeMonthly), store.StatUsers(), logger, job.RecordTypeMonthly)
	if _, err := scheduler.Register("5 0 * * *", statUserJobMonthly); err != nil {
		return err
	}
	trafficFetchJob := job.NewTrafficFetchJob(trafficQueue, serverTrafficService, logger)
	if _, err := scheduler.Register("@every 10s", trafficFetchJob); err != nil {
		return err
	}
	emailJob := job.NewSendEmailJob(notificationQueue, infra.Notifier, logger)
	if _, err := scheduler.Register("@every 10s", emailJob); err != nil {
		return err
	}
	telegramJob := job.NewSendTelegramJob(notificationQueue, infra.Notifier, logger)
	if _, err := scheduler.Register("@every 10s", telegramJob); err != nil {
		return err
	}
	heartbeatJob := job.NewNodeHeartbeatJob(store.Servers(), notificationQueue, store.Settings(), logger)
	if _, err := scheduler.Register("@every 1m", heartbeatJob); err != nil {
		return err
	}
	trafficPeriodResetJob := job.NewTrafficPeriodResetJob(userTrafficService, logger)
	if _, err := scheduler.Register("0 0 0 * * *", trafficPeriodResetJob); err != nil {
		return err
	}
	accessLogCleanupJob := job.NewAccessLogCleanupJob(accessLogService, logger)
	if _, err := scheduler.Register("@every 1h", accessLogCleanupJob); err != nil {
		return err
	}
	scheduler.Start()

	services := api.Services{
		Config:              service.NewConfigService(store.Settings(), i18nManager),
		User:                service.NewUserService(store.Users(), store.Settings(), infra.Hasher),
		UserStat:            userStatService,
		Auth:                service.NewAuthService(store.Users(), store.Settings(), store.LoginLogs(), store.Tokens(), infra.Hasher, infra.Token, infra.RateLimiter, infra.Audit, infra.Cache),
		AdminPath:           service.NewAdminPathService(store.Settings()),
		Install:             installService,
		AdminPlan:           adminPlanService,
		AdminUser:           adminUserService,
		AdminServer:         adminServerService,
		AdminStat:           adminStatService,
		AdminNodeStat:       adminNodeStatService,
		AdminSystem:         adminSystemService,
		AdminSystemSettings: adminSystemSettingsService,
		AdminNotice:         adminNoticeService,
		AdminKnowledge:      adminKnowledgeService,
		UserKnowledge:       userKnowledgeService,
		UserNotice:          userNoticeService,
		ServerAuth:          serverAuthService,
		ServerNode:          serverNodeService,
		Traffic:             serverTrafficService,
		Telemetry:           serverTelemetryService,
		Verify:              verifyService,
		Invite:              inviteService,
		Password:            passwordService,
		Register:            registrationService,
		MailLink:            mailLinkService,
		Comm:                commService,
		Plan:                planService,
		Server:              service.NewServerService(store.Users(), store.Servers(), store.Plans()),
		Subscription:        service.NewSubscriptionService(store.Users(), store.Servers(), store.Settings(), store.Plans(), store.SubscriptionTemplates(), protocolManager, serverTelemetryService, subLogQueue, cfg.Security.SubscribeObfuscation, userServerSelectionService, i18nManager),
		AgentHost:           agentHostService,
		AgentCore:           agentCoreService,
		Forwarding:          forwardingService,
		AccessLog:           accessLogService,
		UserSelection:       userServerSelectionService,
		ShortLink:           shortLinkService,
		TrafficQueue:        trafficQueue,
		SubLogQueue:         subLogQueue,
		I18n:                i18nManager,
	}

	router := api.NewRouter(
		logger,
		services,
		cfg.Metrics,
		api.WithAdminUI(api.AdminUIOptions{
			Enabled:       cfg.UI.Admin.Enabled,
			Dir:           cfg.UI.Admin.Dir,
			BaseURL:       cfg.UI.Admin.BaseURL,
			Title:         cfg.UI.Admin.Title,
			Version:       cfg.UI.Admin.Version,
			Logo:          "https://xboard.io/images/logo.png", // TODO: Add to config
			HiddenModules: cfg.UI.Admin.HiddenModules,
		}),
		api.WithUserUI(api.UserUIOptions{
			Enabled: cfg.UI.User.Enabled,
			Dir:     cfg.UI.User.Dir,
			BaseURL: cfg.UI.User.BaseURL,
			Title:   cfg.UI.User.Title,
		}),
		api.WithInstallUI(api.InstallUIOptions{
			Enabled: cfg.UI.Install.Enabled,
			Dir:     cfg.UI.Install.Dir,
		}),
	)

	server := bootstrap.NewHTTPServer(legacyCfg, router)

	// Start gRPC server if enabled
	var grpcServer *internalgrpc.Server
	if cfg.GRPC.Enabled {
		authInterceptor := interceptor.NewAuthInterceptor(agentHostService)
		agentHandler := handler.NewAgentHandler(
			agentHostService,
			agentService,
			serverTelemetryService,
			serverNodeService,
			userTrafficService,
			forwardingService,
			accessLogService,
			adminSystemSettingsService,
			logger,
		)

		grpcCfg := internalgrpc.Config{
			Address: cfg.GRPC.Addr,
		}
		if cfg.GRPC.TLS.Enabled {
			grpcCfg.TLS = &internalgrpc.TLSConfig{
				Enabled:  true,
				CertFile: cfg.GRPC.TLS.CertFile,
				KeyFile:  cfg.GRPC.TLS.KeyFile,
			}
		}

		var err error
		grpcServer, err = internalgrpc.NewServer(grpcCfg, agentHandler, authInterceptor, logger)
		if err != nil {
			return err
		}

		go func() {
			logger.Info("gRPC server starting", "addr", cfg.GRPC.Addr)
			if err := grpcServer.Start(); err != nil {
				logger.Error("gRPC server failed", "error", err)
				stop()
			}
		}()
	}

	go func() {
		logger.Info("http server starting", "addr", cfg.HTTP.Addr, "env", cfg.Log.Environment)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	stopCtx := scheduler.Stop()
	<-stopCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	// Shutdown gRPC server
	if grpcServer != nil {
		logger.Info("shutting down gRPC server")
		grpcServer.Stop()
	}

	logger.Info("shutting down http server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", "error", err)
	}
	logger.Info("server exited cleanly")
	return nil
}
