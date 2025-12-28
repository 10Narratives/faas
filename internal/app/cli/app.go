package cliapp

import "go.uber.org/zap"

type App struct {
	cfg *Config
	log *zap.Logger
}

func NewApp(cfg *Config, log *zap.Logger) (*App, error) {
	return nil, nil
}
