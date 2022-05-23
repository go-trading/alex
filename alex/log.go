package main

// Инициация уровня логирования в текущем

import (
	"github.com/go-trading/alex"
	"github.com/go-trading/alex/history"
	"github.com/go-trading/alex/tinkoff"
	"go.uber.org/zap"
)

var l *zap.Logger

func init() {
	logger, _ := zap.NewProduction()
	l = logger
}

func initDebugLogger() {
	logger, _ := zap.NewDevelopment()
	l = logger
	alex.SetLogger(l)
	tinkoff.SetLogger(l)
	history.SetLogger(l)
}
