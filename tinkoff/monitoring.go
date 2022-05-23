package tinkoff

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type PrometheusService struct {
	server *http.Server
}

// Start the prometheus service.
func (s *PrometheusService) Start(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{
		MaxRequestsInFlight: 5,
		Timeout:             30 * time.Second,
	}))

	s.server = &http.Server{Addr: addr, Handler: mux}

	go func() {
		l.Debug("Starting prometheus service", zap.String("address", s.server.Addr))
		err := s.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			l.Error("Could not listen", zap.String("address", s.server.Addr), zap.Error(err))
		}
	}()
	return nil
}

// Stop the service gracefully.
func (s *PrometheusService) Stop() error {
	if s.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}
