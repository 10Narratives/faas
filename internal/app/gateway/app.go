package gatewayapp

import (
	"context"
	"fmt"

	grpcsrv "github.com/10Narratives/faas/internal/app/components/grpc/server"
	miniocomp "github.com/10Narratives/faas/internal/app/components/minio"
	natscomp "github.com/10Narratives/faas/internal/app/components/nats"
	pgcomp "github.com/10Narratives/faas/internal/app/components/postgres"
	funcsrv "github.com/10Narratives/faas/internal/services/functions"
	funcapi "github.com/10Narratives/faas/internal/transport/grpc/api/functions"
	healthapi "github.com/10Narratives/faas/internal/transport/grpc/api/health"
	reflectapi "github.com/10Narratives/faas/internal/transport/grpc/api/reflect"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/logging"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/recovery"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/validator"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type App struct {
	cfg *Config
	log *zap.Logger

	stateDB       *pgxpool.Pool
	objectStorage *minio.Client
	taskQueue     *nats.Conn
	grpcServer    *grpcsrv.Component
}

func NewApp(cfg *Config, log *zap.Logger) (*App, error) {
	stateDB, err := pgcomp.NewConnection(context.Background(), cfg.Databases.StateDB.DSN)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to state database: %w", err)
	}
	log.Info("connection to state database established")

	objectStorage, err := miniocomp.NewConnection(context.Background(), cfg.Databases.ObjectStorage.Endpoint,
		cfg.Databases.ObjectStorage.AccessKey, cfg.Databases.ObjectStorage.SecretKey, false)

	if err != nil {
		return nil, fmt.Errorf("cannot connect to object storage: %w", err)
	}
	log.Info("connection to object storage established")

	taskQueue, err := natscomp.NewConnection(cfg.Transport.TaskQueue.URL)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to task queue: %w", err)
	}
	log.Info("connection to task queue established")

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
		cfg:           cfg,
		log:           log,
		grpcServer:    grpcServer,
		stateDB:       stateDB,
		objectStorage: objectStorage,
		taskQueue:     taskQueue,
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
		a.log.Debug("closing connection to state database")
		defer a.log.Info("connection to state database closed")

		a.stateDB.Close()
		return nil
	})

	errGroup.Go(func() error {
		a.log.Debug("closing connection to task queue")
		defer a.log.Info("connection to task queue closed")

		a.taskQueue.Close()
		return nil
	})

	return errGroup.Wait()
}
