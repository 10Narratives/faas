package gatewayapp

import (
	"context"

	grpctr "github.com/10Narratives/faas/internal/transport/grpc"
	funcapi "github.com/10Narratives/faas/internal/transport/grpc/functions"
	opapi "github.com/10Narratives/faas/internal/transport/grpc/operations"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type App struct {
	cfg        *Config
	log        *zap.Logger
	grpcServer *grpctr.Component
}

func NewApp(cfg *Config, log *zap.Logger) (*App, error) {
	grpcServer := grpctr.NewComponent(cfg.Transport.Grpc.Address,
		grpctr.WithServerOptions(),
		grpctr.WithServiceRegistration(
			funcapi.NewRegistration(nil),
			opapi.NewRegistration(nil),
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
