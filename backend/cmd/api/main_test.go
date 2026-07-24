package main

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	values := map[string]string{
		"HTTP_ADDRESS":          ":8080",
		"DATABASE_URL":          "postgres://example",
		"HTTP_SHUTDOWN_TIMEOUT": "5s",
	}

	config, err := loadConfig(func(name string) string { return values[name] })
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if config == nil {
		t.Fatal("loadConfig() result = nil, want configuration")
	}
	if config.address != ":8080" || config.shutdownTimeout != 5*time.Second {
		t.Fatalf("loadConfig() = %#v", config)
	}
}

func TestRunRejectsNilConfiguration(t *testing.T) {
	t.Parallel()

	if err := run(nil); err == nil {
		t.Fatal("run(nil) error = nil, want configuration error")
	}
}

func TestLoadConfigRejectsMissingAndInvalidShutdownTimeout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values map[string]string
	}{
		{
			name: "missing database URL",
			values: map[string]string{
				"HTTP_ADDRESS":          ":8080",
				"HTTP_SHUTDOWN_TIMEOUT": "5s",
			},
		},
		{
			name: "invalid shutdown timeout",
			values: map[string]string{
				"HTTP_ADDRESS":          ":8080",
				"DATABASE_URL":          "postgres://example",
				"HTTP_SHUTDOWN_TIMEOUT": "forever",
			},
		},
		{
			name: "zero shutdown timeout",
			values: map[string]string{
				"HTTP_ADDRESS":          ":8080",
				"DATABASE_URL":          "postgres://example",
				"HTTP_SHUTDOWN_TIMEOUT": "0s",
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			config, err := loadConfig(func(name string) string {
				return test.values[name]
			})
			if err == nil {
				t.Fatal("loadConfig() error = nil, want validation error")
			}
			if config != nil {
				t.Fatalf("loadConfig() result = %#v, want nil on error", config)
			}
		})
	}
}

func TestServeUntilShutdownStopsCanceledServer(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	serverStopped := make(chan struct{})
	var stopOnce sync.Once
	serve := func() error {
		<-serverStopped
		return http.ErrServerClosed
	}
	shutdown := func(ctx context.Context) error {
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			t.Error("shutdown context has no deadline")
		}
		stopOnce.Do(func() { close(serverStopped) })
		return nil
	}

	if err := serveUntilShutdown(
		ctx,
		time.Second,
		serve,
		shutdown,
	); err != nil {
		t.Fatalf("serveUntilShutdown() error = %v", err)
	}
}

func TestServeUntilShutdownReportsListenFailure(t *testing.T) {
	t.Parallel()

	listenError := errors.New("listen failed")
	err := serveUntilShutdown(
		context.Background(),
		time.Second,
		func() error { return listenError },
		func(context.Context) error {
			t.Fatal("shutdown called after listen failure")
			return nil
		},
	)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("serveUntilShutdown() error = %v, want listen failure", err)
	}
	if !errors.Is(err, listenError) {
		t.Fatalf("serveUntilShutdown() error = %v, want wrapped listen error", err)
	}
}

func TestServeUntilShutdownReportsShutdownFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	serveStarted := make(chan struct{})
	releaseServe := make(chan struct{})
	go func() {
		<-serveStarted
		cancel()
	}()
	shutdownError := errors.New("shutdown failed")

	err := serveUntilShutdown(
		ctx,
		time.Second,
		func() error {
			close(serveStarted)
			<-releaseServe
			return http.ErrServerClosed
		},
		func(context.Context) error {
			close(releaseServe)
			return shutdownError
		},
	)
	if !errors.Is(err, shutdownError) {
		t.Fatalf("serveUntilShutdown() error = %v, want wrapped shutdown error", err)
	}
}
