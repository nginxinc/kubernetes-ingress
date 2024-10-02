package log

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/nginxinc/kubernetes-ingress/internal/logger/glog"
	"github.com/nginxinc/kubernetes-ingress/internal/logger/levels"
)

type ctxLogger struct{}

// ContextWithLogger adds logger to context
func ContextWithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxLogger{}, l)
}

// LoggerFromContext returns logger from context
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxLogger{}).(*slog.Logger); ok {
		return l
	}
	return slog.New(glog.New(os.Stdout, nil))
}

// Tracef returns formatted trace log
func Tracef(logger *slog.Logger, format string, args ...interface{}) {
	if !logger.Enabled(context.Background(), levels.LevelTrace) {
		return
	}
	logger.Log(context.Background(), levels.LevelTrace, fmt.Sprintf(format, args...))
}

// Trace returns raw trace log
func Trace(logger *slog.Logger, args ...interface{}) {
	if !logger.Enabled(context.Background(), levels.LevelTrace) {
		return
	}
	logger.Log(context.Background(), levels.LevelTrace, fmt.Sprint(args...))
}

// Debugf returns formatted trace log
func Debugf(logger *slog.Logger, format string, args ...interface{}) {
	if !logger.Enabled(context.Background(), levels.LevelDebug) {
		return
	}
	logger.Debug(fmt.Sprintf(format, args...))
}

// Debug returns raw trace log
func Debug(logger *slog.Logger, args ...interface{}) {
	if !logger.Enabled(context.Background(), levels.LevelDebug) {
		return
	}
	logger.Debug(fmt.Sprint(args...))
}

// Infof returns formatted trace log
func Infof(logger *slog.Logger, format string, args ...interface{}) {
	if !logger.Enabled(context.Background(), levels.LevelInfo) {
		return
	}
	logger.Info(fmt.Sprintf(format, args...))
}

// Info returns raw trace log
func Info(logger *slog.Logger, args ...interface{}) {
	if !logger.Enabled(context.Background(), levels.LevelInfo) {
		return
	}
	logger.Info(fmt.Sprint(args...))
}

// Warnf returns formatted trace log
func Warnf(logger *slog.Logger, format string, args ...interface{}) {
	if !logger.Enabled(context.Background(), levels.LevelWarning) {
		return
	}
	logger.Warn(fmt.Sprintf(format, args...))
}

// Warn returns raw trace log
func Warn(logger *slog.Logger, args ...interface{}) {
	if !logger.Enabled(context.Background(), levels.LevelWarning) {
		return
	}
	logger.Warn(fmt.Sprint(args...))
}

// Errorf returns formatted trace log
func Errorf(logger *slog.Logger, format string, args ...interface{}) {
	if !logger.Enabled(context.Background(), levels.LevelError) {
		return
	}
	logger.Error(fmt.Sprintf(format, args...))
}

// Error returns raw trace log
func Error(logger *slog.Logger, args ...interface{}) {
	if !logger.Enabled(context.Background(), levels.LevelError) {
		return
	}
	logger.Error(fmt.Sprint(args...))
}

// Fatalf returns formatted trace log
func Fatalf(logger *slog.Logger, format string, args ...interface{}) {
	if !logger.Enabled(context.Background(), levels.LevelFatal) {
		return
	}
	logger.Log(context.Background(), levels.LevelFatal, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// Fatal returns raw trace log
func Fatal(logger *slog.Logger, args ...interface{}) {
	if !logger.Enabled(context.Background(), levels.LevelFatal) {
		return
	}
	logger.Log(context.Background(), levels.LevelFatal, fmt.Sprint(args...))
	os.Exit(1)
}
