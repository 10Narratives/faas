package agentapp

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	runtime "github.com/10Narratives/faas/internal/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type App struct {
	cfg     *Config
	log     *zap.Logger
	manager *runtime.Manager
}

func NewApp(cfg *Config, log *zap.Logger) (*App, error) {
	managerCfg := &runtime.ManagerConfig{
		MaxInstances:     100,
		InstanceLifetime: 2 * time.Minute,
		ColdStart:        200 * time.Millisecond,
		NATSURL:          cfg.UnifiedStorage.URL,
		PodName:          os.Getenv("POD_NAME"),
		MaxAckPending:    25,
		AckWait:          15 * time.Second,
		MaxDeliver:       5,
		Backoff: []time.Duration{
			100 * time.Millisecond,
			500 * time.Millisecond,
			2 * time.Second,
			5 * time.Second,
			10 * time.Second,
		},
	}
	manager, err := runtime.NewManager(log, managerCfg)
	if err != nil {
		return nil, fmt.Errorf("create runtime manager: %w", err)
	}

	return &App{
		cfg:     cfg,
		log:     log,
		manager: manager,
	}, nil
}

func (a *App) Startup(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		http.Handle("/metrics", promhttp.Handler())
		return http.ListenAndServe(":8080", nil)
	})

	errGroup.Go(func() error {
		a.log.Info("start function manager")
		return a.manager.Run(ctx)
	})

	return errGroup.Wait()
}

func (a *App) Shutdown(ctx context.Context) error {
	errGroup, ctx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		if a.manager != nil {
			if err := a.manager.Stop(ctx); err != nil {
				a.log.Warn("manager stop failed", zap.Error(err))
			}
		}
		return nil
	})

	return errGroup.Wait()
}
