package main

import (
	"net/http"
	"time"

	"github.com/charmbracelet/log"
)

type loggerTransport struct {
	transport http.RoundTripper
	logger    *log.Logger
}

func (l *loggerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	l.logger.Debug("HTTP Request",
		"method", req.Method,
		"url", req.URL.String(),
	)

	startTime := time.Now()
	resp, err := l.transport.RoundTrip(req)
	if err != nil {
		l.logger.Error("HTTP Request failed", "error", err)
		return nil, err
	}
	duration := time.Since(startTime)

	l.logger.Debug("HTTP Response",
		"status", resp.Status,
		"duration", duration,
		"url", req.URL.String(),
		"method", req.Method,
	)

	return resp, nil
}

func newLoggingTransport(transport http.RoundTripper, logger *log.Logger) http.RoundTripper {
	return &loggerTransport{transport: transport, logger: logger}
}
