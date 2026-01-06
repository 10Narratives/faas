package gatewayapp

import (
	"context"
	"fmt"

	grpcsrv "github.com/10Narratives/faas/internal/app/components/grpc/server"
	natscomp "github.com/10Narratives/faas/internal/app/components/nats"
	healthapi "github.com/10Narratives/faas/internal/transport/grpc/dev/health"
	reflectapi "github.com/10Narratives/faas/internal/transport/grpc/dev/reflect"

	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/logging"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/recovery"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/validator"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type App struct {
	cfg *Config
	log *zap.Logger

	unifiedStorage *nats.Conn
	grpcServer     *grpcsrv.Component
}

func NewApp(cfg *Config, log *zap.Logger) (*App, error) {
	unifiedStorage, err := natscomp.NewConnection(cfg.UnifiedStorage.URL)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to unified storage: %w", err)
	}
	log.Info("connection to unified storage established")

	grpcServer := grpcsrv.NewComponent(cfg.Server.Grpc.Address,
		grpcsrv.WithServerOptions(
			grpc.ChainUnaryInterceptor(
				recovery.NewUnaryServerInterceptor(),
				logging.NewUnaryServerInterceptor(log),
				validator.NewUnaryServerInterceptor(),
			),
			grpc.ChainStreamInterceptor(
				recovery.NewStreamServerInterceptor(),
				logging.NewStreamServerInterceptor(log),
				validator.NewStreamServerInterceptor(),
			),
		),
		grpcsrv.WithServiceRegistration(
			healthapi.NewRegistration(),
			reflectapi.NewRegistration(),
		),
	)

	return &App{
		cfg:            cfg,
		log:            log,
		grpcServer:     grpcServer,
		unifiedStorage: unifiedStorage,
	}, nil
}

func (a *App) Startup(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		a.log.Debug("starting gRPC server")
		defer a.log.Info("gRPC server ready to accept requests")

		return a.grpcServer.Startup(ctx)
	})

	return errGroup.Wait()
}

func (a *App) Shutdown(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		a.log.Debug("stopping gRPC server")
		defer a.log.Info("gRPC server stopped")

		return a.grpcServer.Shutdown(ctx)
	})

	errGroup.Go(func() error {
		a.log.Debug("closing connection to task queue")
		defer a.log.Info("connection to task queue closed")

		a.unifiedStorage.Close()
		return nil
	})

	return errGroup.Wait()
}
