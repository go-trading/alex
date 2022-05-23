package history

import (
	"go.uber.org/zap"
)

var l *zap.Logger

func init() {
	logger, _ := zap.NewProduction()
	l = logger
}

func SetLogger(logger *zap.Logger) {
	l = logger
}
