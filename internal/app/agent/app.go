package agentapp

import (
	"context"
	"fmt"
	"time"

	natscomp "github.com/10Narratives/faas/internal/app/components/nats"
	funcrepo "github.com/10Narratives/faas/internal/repositories/functions"
	taskrepo "github.com/10Narratives/faas/internal/repositories/tasks"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type App struct {
	cfg *Config
	log *zap.Logger

	unifiedStorage *natscomp.UnifiedStorage

	taskRepo *taskrepo.Repository

	funcMeta *funcrepo.MetadataRepository
	funcObj  *funcrepo.ObjectRepository
}

func NewApp(cfg *Config, log *zap.Logger) (*App, error) {
	unifiedStorage, err := natscomp.NewUnifiedStorage(cfg.UnifiedStorage.URL)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to unified storage: %w", err)
	}
	log.Info("connection to unified storage established")

	taskRepo := taskrepo.NewRepository(unifiedStorage.TaskMeta)
	funcMetaRepo := funcrepo.NewMetadataRepository(unifiedStorage.FuncMeta)
	funcObjRepo := funcrepo.NewObjectRepository(unifiedStorage.FuncObj)

	return &App{
		cfg:            cfg,
		log:            log,
		unifiedStorage: unifiedStorage,
		taskRepo:       taskRepo,
		funcMeta:       funcMetaRepo,
		funcObj:        funcObjRepo,
	}, nil
}

func (a *App) Startup(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		a.log.Info("agent online")
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		select {
		case <-ticker.C:
			a.log.Info("work done")
			return nil
		case <-ctx.Done():
			a.log.Info("work canceled")
			return nil
		}
	})

	return errGroup.Wait()
}

func (a *App) Shutdown(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		a.log.Debug("closing connection to unified storage")
		defer a.log.Info("connection to task unified storage")

		a.unifiedStorage.Conn.Close()
		return nil
	})

	return errGroup.Wait()
}
