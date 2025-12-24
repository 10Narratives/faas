package logutils

import (
	"go.uber.org/zap"
)

func NewLogger(env string) (*zap.Logger, error) {
	switch env {
	case "prod":
		return zap.NewProduction()
	default:
		return zap.NewDevelopment()
	}
}
