package agentapp

import (
	"context"
	"fmt"
	"net"
	"net/http"

	funcrepo "github.com/10Narratives/faas/internal/repository/functions"
	"github.com/10Narratives/faas/internal/services/runtime"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type App struct {
	objectStorage *minio.Client

	functionRepository *funcrepo.Repository
	runtime            *runtime.Runtime

	cfg *Config
	log *zap.Logger
}

func NewApp(cfg *Config, log *zap.Logger) (*App, error) {
	log.Info("object storage connection config",
		zap.String("endpoint", cfg.ObjectStorage.Endpoint),
		zap.String("user", cfg.ObjectStorage.User),
		zap.Bool("use_ssl", cfg.ObjectStorage.UseSSL),
	)

	host, port, err := net.SplitHostPort(cfg.ObjectStorage.Endpoint)
	if err != nil {
		host = cfg.ObjectStorage.Endpoint
		port = "9000"
	}

	objectStorage, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.ObjectStorage.User, cfg.ObjectStorage.Password, ""),
		Secure: cfg.ObjectStorage.UseSSL,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial(network, net.JoinHostPort(host, port))
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("cannot create new minio client: %w", err)
	}

	functionRepository, err := funcrepo.NewRepository(objectStorage, cfg.ObjectStorage.FunctionsBucketName)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize function repo: %w", err)
	}

	runtime, err := runtime.NewRuntime(functionRepository)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize runtime: %w", err)
	}

	return &App{
		objectStorage:      objectStorage,
		functionRepository: functionRepository,
		runtime:            runtime,

		cfg: cfg,
		log: log,
	}, nil
}

func (a *App) Startup(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	a.log.Info("start running function")
	result, _ := a.runtime.RunFunction(ctx, "hello.py")
	a.log.Info("result", zap.ByteString("result", result))
	a.log.Info("stop running function")

	return errGroup.Wait()
}

func (a *App) Shutdown(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)
	return errGroup.Wait()
}
