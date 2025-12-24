package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	agentapp "github.com/10Narratives/faas/internal/app/agent"
	configutils "github.com/10Narratives/faas/pkg/config"
	errorutils "github.com/10Narratives/faas/pkg/errors"
	logutils "github.com/10Narratives/faas/pkg/logging"
)

func main() {
	path := flag.String("config", "", "path to configuration file")
	env := flag.String("env", "", "launch environment")

	flag.Parse()

	cfg := errorutils.Must(readConfig(*path))
	log := errorutils.Must(logutils.NewLogger(*env))

	app := errorutils.Must(agentapp.NewApp(cfg, log))

	startupContext, startupCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	errorutils.Try(app.Startup(startupContext))

	<-startupContext.Done()
	startupCancel()

	shutdownContext, shutdownCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	errorutils.Try(app.Shutdown(shutdownContext))

	shutdownCancel()
}

func readConfig(path string) (*agentapp.Config, error) {
	if path == "" {
		return configutils.ReadFromEnv[agentapp.Config]()
	}
	return configutils.ReadFromFile[agentapp.Config](path)
}
