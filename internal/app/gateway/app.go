package gatewayapp

import (
	"context"

	grpcsrv "github.com/10Narratives/faas/internal/app/components/grpc/server"
	funcsrv "github.com/10Narratives/faas/internal/services/functions"
	funcapi "github.com/10Narratives/faas/internal/transport/grpc/api/functions"
	healthapi "github.com/10Narratives/faas/internal/transport/grpc/api/health"
	reflectapi "github.com/10Narratives/faas/internal/transport/grpc/api/reflect"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/logging"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/recovery"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/validator"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type App struct {
	cfg        *Config
	log        *zap.Logger
	grpcServer *grpcsrv.Component
}

func NewApp(cfg *Config, log *zap.Logger) (*App, error) {
	functionService, err := funcsrv.NewService()
	if err != nil {
		return nil, err
	}

	grpcServer := grpcsrv.NewComponent(cfg.Transport.Grpc.Address,
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
			funcapi.NewRegistration(functionService),
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
