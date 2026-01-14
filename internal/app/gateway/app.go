package gatewayapp

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	grpcsrv "github.com/10Narratives/faas/internal/app/components/grpc/server"
	natscomp "github.com/10Narratives/faas/internal/app/components/nats"
	funcrepo "github.com/10Narratives/faas/internal/repositories/functions"
	taskrepo "github.com/10Narratives/faas/internal/repositories/tasks"
	funcsrv "github.com/10Narratives/faas/internal/services/functions"
	tasksrv "github.com/10Narratives/faas/internal/services/tasks"
	funcapi "github.com/10Narratives/faas/internal/transport/grpc/api/functions"
	taskapi "github.com/10Narratives/faas/internal/transport/grpc/api/tasks"
	healthapi "github.com/10Narratives/faas/internal/transport/grpc/dev/health"
	reflectapi "github.com/10Narratives/faas/internal/transport/grpc/dev/reflect"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/logging"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/recovery"
	"github.com/10Narratives/faas/internal/transport/grpc/interceptors/validator"
	grpcprom "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

type App struct {
	cfg            *Config
	log            *zap.Logger
	unifiedStorage *natscomp.UnifiedStorage
	taskRepo       *taskrepo.Repository
	taskPub        *taskrepo.Publisher
	funcMeta       *funcrepo.MetadataRepository
	funcObj        *funcrepo.ObjectRepository
	grpcServer     *grpcsrv.Component
	metricsServer  *http.Server
}

func NewApp(cfg *Config, log *zap.Logger) (*App, error) {
	srvMetrics := grpcprom.NewServerMetrics()
	prometheus.MustRegister(srvMetrics)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	metricsSrv := &http.Server{
		Addr:    cfg.Server.Metrics.Address,
		Handler: mux,
	}

	unifiedStorage, err := natscomp.NewUnifiedStorage(cfg.UnifiedStorage.URL)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to unified storage: %w", err)
	}
	log.Info("connection to unified storage established")

	taskRepo := taskrepo.NewRepository(unifiedStorage.TaskMeta)
	taskPub := taskrepo.NewPublisher(unifiedStorage.JS)
	funcMetaRepo := funcrepo.NewMetadataRepository(unifiedStorage.FuncMeta)
	funcObjRepo := funcrepo.NewObjectRepository(unifiedStorage.FuncObj)

	taskService := tasksrv.NewService(taskRepo, taskPub)
	funcService := funcsrv.NewService(funcMetaRepo, funcObjRepo, taskService)

	grpcServer := grpcsrv.NewComponent(cfg.Server.Grpc.Address,
		grpcsrv.WithServerOptions(
			grpc.ChainUnaryInterceptor(
				recovery.NewUnaryServerInterceptor(),
				logging.NewUnaryServerInterceptor(log),
				validator.NewUnaryServerInterceptor(),
				srvMetrics.UnaryServerInterceptor(),
			),
			grpc.ChainStreamInterceptor(
				recovery.NewStreamServerInterceptor(),
				logging.NewStreamServerInterceptor(log),
				validator.NewStreamServerInterceptor(),
				srvMetrics.StreamServerInterceptor(),
			),
		),
		grpcsrv.WithServiceRegistration(
			healthapi.NewRegistration(),
			reflectapi.NewRegistration(),
			taskapi.NewRegistration(taskService),
			funcapi.NewRegistration(funcService),
		),
	)

	srvMetrics.InitializeMetrics(grpcServer.Server)

	return &App{
		cfg:            cfg,
		log:            log,
		grpcServer:     grpcServer,
		unifiedStorage: unifiedStorage,
		taskRepo:       taskRepo,
		taskPub:        taskPub,
		funcMeta:       funcMetaRepo,
		funcObj:        funcObjRepo,
		metricsServer:  metricsSrv,
	}, nil
}

func (a *App) Startup(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		a.log.Debug("starting metrics http server")
		err := a.metricsServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})

	errGroup.Go(func() error {
		a.log.Debug("starting gRPC server")
		return a.grpcServer.Startup(ctx)
	})

	return errGroup.Wait()
}

func (a *App) Shutdown(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		a.log.Debug("stopping metrics http server")
		return a.metricsServer.Shutdown(ctx)
	})

	errGroup.Go(func() error {
		a.log.Debug("stopping gRPC server")
		return a.grpcServer.Shutdown(ctx)
	})

	errGroup.Go(func() error {
		a.unifiedStorage.Conn.Close()
		return nil
	})

	return errGroup.Wait()
}
