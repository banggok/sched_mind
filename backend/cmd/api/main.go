package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/banggok/sched_mind/backend/internal/httpapi"
	roleapplication "github.com/banggok/sched_mind/backend/internal/roles/application"
	rolegormrepo "github.com/banggok/sched_mind/backend/internal/roles/infrastructure/gormrepo"
	rolepostgres "github.com/banggok/sched_mind/backend/internal/roles/infrastructure/postgres"
	teammemberapplication "github.com/banggok/sched_mind/backend/internal/teammembers/application"
	teammembergormrepo "github.com/banggok/sched_mind/backend/internal/teammembers/infrastructure/gormrepo"
	teammemberpostgres "github.com/banggok/sched_mind/backend/internal/teammembers/infrastructure/postgres"
)

func main() {
	config, err := loadConfig(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}
	if err := run(config); err != nil {
		log.Fatal(err)
	}
}

type configuration struct {
	address         string
	databaseURL     string
	shutdownTimeout time.Duration
}

func loadConfig(environment func(string) string) (*configuration, error) {
	address, err := requiredEnvironment(environment, "HTTP_ADDRESS")
	if err != nil {
		return nil, err
	}
	databaseURL, err := requiredEnvironment(environment, "DATABASE_URL")
	if err != nil {
		return nil, err
	}
	rawShutdownTimeout, err := requiredEnvironment(
		environment,
		"HTTP_SHUTDOWN_TIMEOUT",
	)
	if err != nil {
		return nil, err
	}
	shutdownTimeout, err := time.ParseDuration(rawShutdownTimeout)
	if err != nil || shutdownTimeout <= 0 {
		return nil, fmt.Errorf(
			"HTTP_SHUTDOWN_TIMEOUT must be a positive duration",
		)
	}

	return &configuration{
		address:         address,
		databaseURL:     databaseURL,
		shutdownTimeout: shutdownTimeout,
	}, nil
}

func requiredEnvironment(
	environment func(string) string,
	name string,
) (string, error) {
	value := environment(name)
	if value == "" {
		return "", fmt.Errorf("%s environment variable is required", name)
	}
	return value, nil
}

func run(config *configuration) (runError error) {
	if config == nil {
		return errors.New("application configuration is nil")
	}

	database, err := rolepostgres.Open(config.databaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	if database == nil {
		return errors.New("database connection is nil")
	}
	sqlDatabase, err := database.DB()
	if err != nil {
		return fmt.Errorf("access database connection: %w", err)
	}
	if sqlDatabase == nil {
		return errors.New("SQL database connection is nil")
	}
	defer func() {
		if err := sqlDatabase.Close(); err != nil {
			runError = errors.Join(
				runError,
				fmt.Errorf("close database connection: %w", err),
			)
		}
	}()

	if err := rolepostgres.Migrate(database); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	if err := teammemberpostgres.Migrate(database); err != nil {
		return fmt.Errorf("migrate team member database: %w", err)
	}

	roleRepository := rolegormrepo.New(database)
	roleService := roleapplication.NewService(roleRepository)
	teamMemberRepository := teammembergormrepo.New(database)
	teamMemberService := teammemberapplication.NewService(teamMemberRepository)

	server := &http.Server{
		Addr:    config.address,
		Handler: httpapi.NewRouter(roleService, teamMemberService),
	}

	signalContext, stopSignals := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stopSignals()

	log.Printf("API listening on %s", config.address)
	return serveUntilShutdown(
		signalContext,
		config.shutdownTimeout,
		server.ListenAndServe,
		server.Shutdown,
	)
}

func serveUntilShutdown(
	ctx context.Context,
	timeout time.Duration,
	serve func() error,
	shutdown func(context.Context) error,
) error {
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- serve()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve API: %w", err)
	case <-ctx.Done():
		log.Print("API shutdown requested")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(
		context.Background(),
		timeout,
	)
	defer cancelShutdown()

	if err := shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shutdown API: %w", err)
	}

	if err := <-serverErrors; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve API during shutdown: %w", err)
	}

	log.Print("API shutdown complete")
	return nil
}
