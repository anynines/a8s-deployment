package log

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

func NewWithNames(names ...string) logr.Logger {
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	logger := zapr.NewLogger(zapLogger)
	for _, n := range names {
		logger = logger.WithName(n)
	}

	return logger
}
