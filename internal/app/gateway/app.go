package gatewayapp

import (
	"context"

	grpctr "github.com/10Narratives/faas/internal/transport/grpc"
	healthapi "github.com/10Narratives/faas/internal/transport/grpc/health"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/logging"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/recovery"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/validator"
	reflectapi "github.com/10Narratives/faas/internal/transport/grpc/reflection"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type App struct {
	cfg        *Config
	log        *zap.Logger
	grpcServer *grpctr.Component
}

func NewApp(cfg *Config, log *zap.Logger) (*App, error) {
	grpcServer := grpctr.NewComponent(cfg.Transport.Grpc.Address,
		grpctr.WithServerOptions(
			grpc.ChainUnaryInterceptor(
				recovery.NewUnaryServerInterceptor(),
				logging.NewUnaryServerInterceptor(log),
				validator.NewUnaryServerInterceptor(),
			),
		),
		grpctr.WithServiceRegistration(
			healthapi.NewRegistration(),
			reflectapi.NewRegistration(),
		),
	)

	return &App{
		cfg:        cfg,
		log:        log,
		grpcServer: grpcServer,
	}, nil
}

func (a *App) Startup(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		a.log.Info("starting gRPC server")
		return a.grpcServer.Startup(ctx)
	})

	return errGroup.Wait()
}

func (a *App) Shutdown(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		a.log.Info("stopping gRPC server")
		return a.grpcServer.Shutdown(ctx)
	})

	return errGroup.Wait()
}
