package agentapp

import (
	"context"
	"fmt"
	"time"

	miniocomp "github.com/10Narratives/faas/internal/app/components/minio"
	natscomp "github.com/10Narratives/faas/internal/app/components/nats"
	pgcomp "github.com/10Narratives/faas/internal/app/components/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type App struct {
	cfg           *Config
	log           *zap.Logger
	stateDB       *pgxpool.Pool
	objectStorage *minio.Client
	taskQueue     *nats.Conn
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

	return &App{
		cfg:           cfg,
		log:           log,
		stateDB:       stateDB,
		objectStorage: objectStorage,
		taskQueue:     taskQueue,
	}, nil
}

func (a *App) Startup(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		a.log.Info("agent online")
		time.Sleep(1 * time.Minute)
		return nil
	})

	return errGroup.Wait()
}

func (a *App) Shutdown(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)
	return errGroup.Wait()
}
