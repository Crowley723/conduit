package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Crowley723/conduit/internal/authentication"
	"github.com/Crowley723/conduit/internal/config"
	"github.com/Crowley723/conduit/internal/distributed"
	"github.com/Crowley723/conduit/internal/jobs"
	"github.com/Crowley723/conduit/internal/middlewares"
	"github.com/Crowley723/conduit/internal/services/certificate"
	"github.com/Crowley723/conduit/internal/storage"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	cfg         *config.Config
	logger      *slog.Logger
	appCtx      *middlewares.AppContext
	httpServer  *http.Server
	debugServer *http.Server
	election    *distributed.Election
	jobManager  *jobs.JobManager
	ctx         *context.Context
	cancel      context.CancelFunc
}

func New(cfg *config.Config, logger *slog.Logger) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	sessionManager, err := authentication.NewSessionManager(logger, cfg)
	if err != nil {
		cancel()
		return nil, err
	}

	oidcProvider, err := authentication.NewRealOIDCProvider(ctx, cfg.OIDC)
	if err != nil {
		cancel()
		return nil, err
	}

	var election *distributed.Election
	if cfg.Distributed != nil && cfg.Distributed.Enabled {
		var client *redis.Client

		if cfg.Redis.Sentinel != nil {
			logger.Info("connecting to redis via sentinel",
				"master", cfg.Redis.Sentinel.MasterName,
				"sentinels", cfg.Redis.Sentinel.SentinelAddresses)

			client = redis.NewFailoverClient(&redis.FailoverOptions{
				MasterName:       cfg.Redis.Sentinel.MasterName,
				SentinelAddrs:    cfg.Redis.Sentinel.SentinelAddresses,
				SentinelPassword: cfg.Redis.Sentinel.SentinelPassword,
				Password:         cfg.Redis.Password,
				DB:               cfg.Redis.LeaderIndex,
				MinIdleConns:     2,
			})
		} else {
			client = redis.NewClient(&redis.Options{
				Addr:         cfg.Redis.Address,
				Password:     cfg.Redis.Password,
				DB:           cfg.Redis.LeaderIndex,
				MinIdleConns: 2,
			})
		}

		hostname := os.Getenv("HOSTNAME")
		if hostname == "" {
			hostname = uuid.New().String()
		}

		election = &distributed.Election{
			Redis:      client,
			InstanceID: hostname,
			TTL:        cfg.Distributed.TTL,
		}
	}

	var database storage.Provider
	if cfg.Storage != nil {
		dbProvider, err := storage.NewStorageProvider(ctx, cfg)
		if err != nil {
			logger.Error("failed to initialize database provider", "error", err)
			cancel()
			return nil, err
		}

		logger.Debug("Running database migrations")
		if err := dbProvider.RunMigrations(ctx); err != nil {
			logger.Error("failed to run database migrations", "error", err)
			cancel()
			return nil, err
		}

		if err := dbProvider.EnsureSystemUser(ctx, logger); err != nil {
			cancel()
			return nil, fmt.Errorf("failed to ensure system user: %w", err)
		}
		logger.Debug("Database Migrations Completed")

		database = dbProvider
	}

	var certProvider certificate.Provider
	if cfg.MTLS.Enabled {
		if cfg.MTLS.Kubernetes != nil && cfg.MTLS.Kubernetes.Enabled {
			certProvider, err = certificate.NewKubernetesClient(ctx, cfg, logger)
			if err != nil {
				logger.Error("failed to initialize kubernetes client", "error", err)
				cancel()
				return nil, err
			}
		} else {

		}
	}

	appCtx := middlewares.NewAppContext(ctx, cfg, logger, sessionManager, oidcProvider, database, certProvider)

	jobManager := jobs.NewJobManager(election, logger)

	if cfg.MTLS.Enabled {
		certificateCreationJob := jobs.NewCertificateCreationJob(appCtx, cfg.MTLS.BackgroundJobConfig.ApprovedCertificatePollingInterval)
		jobManager.Register(certificateCreationJob)

		certificateIssuedJob := jobs.NewCertificateIssuedStatusJob(appCtx, cfg.MTLS.BackgroundJobConfig.IssuedCertificatePollingInterval)
		jobManager.Register(certificateIssuedJob)
	}

	router := setupRouter(appCtx)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: router,
	}

	var debugServer *http.Server
	if cfg.Server.Debug != nil && cfg.Server.Debug.Enabled {
		debugRouter := setupDebugRouter()
		debugServer = &http.Server{
			Addr:    fmt.Sprintf("%s:%d", cfg.Server.Debug.Host, cfg.Server.Debug.Port),
			Handler: debugRouter,
		}

	}

	return &Server{
		cfg:         cfg,
		logger:      logger,
		appCtx:      appCtx,
		httpServer:  server,
		debugServer: debugServer,
		election:    election,
		jobManager:  jobManager,
		ctx:         &ctx,
		cancel:      cancel,
	}, nil
}

func (s *Server) Start() error {
	if s.election != nil {
		go s.election.Start(*s.appCtx)
	}

	s.jobManager.Start(*s.appCtx)

	router := setupRouter(s.appCtx)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.Server.Port),
		Handler: router,
	}

	go func() {
		if s.cfg.Distributed != nil && s.cfg.Distributed.Enabled {
			s.logger.Info(fmt.Sprintf("Server listening for http at address %s", server.Addr), "instance", s.election.InstanceID)
		} else {
			s.logger.Info(fmt.Sprintf("Server listening for http at address %s", server.Addr))
		}
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("Server failed to start", "error", err)
			s.cancel()
		}
	}()

	if s.cfg.Server.Debug != nil && s.cfg.Server.Debug.Enabled {
		go func() {
			s.logger.Info("Metrics server starting", "address", fmt.Sprintf("%s:%d", s.cfg.Server.Debug.Host, s.cfg.Server.Debug.Port))
			if err := s.debugServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				s.logger.Error("Metrics server failed to start", "error", err)
				s.cancel()
			}
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:

	case <-s.appCtx.Done():
		s.logger.Info("Context canceled")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	s.jobManager.Shutdown(shutdownCtx)

	if err := server.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("Server forced to shutdown", "error", err)
		return err
	}

	if s.debugServer != nil && s.cfg.Server.Debug.Enabled {
		if err := s.debugServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Debug server forced to shutdown", "error", err)
		}
	}

	s.logger.Info("Server Existed")
	return nil
}
