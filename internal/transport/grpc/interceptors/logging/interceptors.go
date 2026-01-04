package logging

import (
	"context"

	l "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func NewUnaryServerInterceptor(log *zap.Logger, opts ...l.Option) grpc.UnaryServerInterceptor {
	return l.UnaryServerInterceptor(NewLoggerFunc(log), opts...)
}

func NewStreamServerInterceptor(log *zap.Logger, opts ...l.Option) grpc.StreamServerInterceptor {
	return l.StreamServerInterceptor(NewLoggerFunc(log), opts...)
}

func NewLoggerFunc(log *zap.Logger) l.Logger {
	return l.LoggerFunc(func(ctx context.Context, level l.Level, msg string, fields ...any) {
		logger := log.WithOptions(zap.AddCallerSkip(3)).Sugar()

		switch level {
		case l.LevelDebug:
			logger.Debugw(msg, fields...)
		case l.LevelInfo:
			logger.Infow(msg, fields...)
		case l.LevelWarn:
			logger.Warnw(msg, fields...)
		case l.LevelError:
			logger.Errorw(msg, fields...)
		default:
			logger.Infow(msg, fields...)
		}
	})
}
