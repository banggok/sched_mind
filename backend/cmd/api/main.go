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

	capacityoverrideapplication "github.com/banggok/sched_mind/backend/internal/capacityoverrides/application"
	capacityoverridegormrepo "github.com/banggok/sched_mind/backend/internal/capacityoverrides/infrastructure/gormrepo"
	dependencyapplication "github.com/banggok/sched_mind/backend/internal/dependencies/application"
	dependencygormrepo "github.com/banggok/sched_mind/backend/internal/dependencies/infrastructure/gormrepo"
	"github.com/banggok/sched_mind/backend/internal/httpapi"
	portfolioapplication "github.com/banggok/sched_mind/backend/internal/portfolio/application"
	portfoliogormrepo "github.com/banggok/sched_mind/backend/internal/portfolio/infrastructure/gormrepo"
	projectapplication "github.com/banggok/sched_mind/backend/internal/projects/application"
	projectgormrepo "github.com/banggok/sched_mind/backend/internal/projects/infrastructure/gormrepo"
	publicholidayapplication "github.com/banggok/sched_mind/backend/internal/publicholidays/application"
	publicholidaygormrepo "github.com/banggok/sched_mind/backend/internal/publicholidays/infrastructure/gormrepo"
	roleapplication "github.com/banggok/sched_mind/backend/internal/roles/application"
	rolegormrepo "github.com/banggok/sched_mind/backend/internal/roles/infrastructure/gormrepo"
	rolepostgres "github.com/banggok/sched_mind/backend/internal/roles/infrastructure/postgres"
	schedulingapplication "github.com/banggok/sched_mind/backend/internal/scheduling/application"
	schedulinggormrepo "github.com/banggok/sched_mind/backend/internal/scheduling/infrastructure/gormrepo"
	"github.com/banggok/sched_mind/backend/internal/shared/identity"
	sprintapplication "github.com/banggok/sched_mind/backend/internal/sprints/application"
	sprintgormrepo "github.com/banggok/sched_mind/backend/internal/sprints/infrastructure/gormrepo"
	teammemberapplication "github.com/banggok/sched_mind/backend/internal/teammembers/application"
	teammembergormrepo "github.com/banggok/sched_mind/backend/internal/teammembers/infrastructure/gormrepo"
	teammemberpostgres "github.com/banggok/sched_mind/backend/internal/teammembers/infrastructure/postgres"
	wbsapplication "github.com/banggok/sched_mind/backend/internal/wbs/application"
	wbsgormrepo "github.com/banggok/sched_mind/backend/internal/wbs/infrastructure/gormrepo"
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
	location        *time.Location
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
	rawTimezone, err := requiredEnvironment(environment, "APP_TIMEZONE")
	if err != nil {
		return nil, err
	}
	location, err := time.LoadLocation(rawTimezone)
	if err != nil {
		return nil, fmt.Errorf("APP_TIMEZONE must be a valid IANA timezone: %w", err)
	}

	return &configuration{
		address:         address,
		databaseURL:     databaseURL,
		shutdownTimeout: shutdownTimeout,
		location:        location,
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
	schedulingRepository := schedulinggormrepo.New(database)
	schedulingService := schedulingapplication.NewService(schedulingRepository)
	if err := teammemberpostgres.MigrateWithPostStep(database, schedulingService.RecalculateActiveProjects); err != nil {
		return fmt.Errorf("migrate team member database: %w", err)
	}

	roleRepository := rolegormrepo.New(database)
	roleService := roleapplication.NewService(roleRepository)
	teamMemberRepository := teammembergormrepo.New(database)
	capacityOverrideRepository := capacityoverridegormrepo.New(database)
	publicHolidayRepository := publicholidaygormrepo.New(database)
	if config.location == nil {
		return errors.New("application timezone is nil")
	}
	teamMemberService := teammemberapplication.NewServiceWithScheduler(teamMemberRepository, schedulingService)
	publicHolidayService := publicholidayapplication.NewServiceWithScheduler(
		publicHolidayRepository,
		schedulingService,
		func() time.Time { return time.Now().In(config.location) },
	)
	capacityOverrideService := capacityoverrideapplication.NewServiceWithScheduler(capacityOverrideRepository, schedulingService)
	projectRepository := projectgormrepo.New(database)
	projectService := projectapplication.NewService(projectRepository, schedulingService)
	wbsRepository := wbsgormrepo.New(database)
	wbsService := wbsapplication.NewServiceWithDependencies(
		wbsRepository,
		schedulingService,
		func() time.Time { return time.Now().In(config.location) },
		identity.NewUUID,
	)
	dependencyRepository := dependencygormrepo.New(database)
	dependencyService := dependencyapplication.NewService(dependencyRepository, schedulingService)
	portfolioRepository := portfoliogormrepo.New(database)
	portfolioService := portfolioapplication.NewService(portfolioRepository)
	sprintRepository := sprintgormrepo.New(database)
	sprintService := sprintapplication.NewService(sprintRepository)

	server := &http.Server{
		Addr:    config.address,
		Handler: httpapi.NewRouter(roleService, teamMemberService, capacityOverrideService, publicHolidayService, projectService, wbsService, dependencyService, portfolioService, sprintService),
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
