package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"syscall"

	"github.com/USA-RedDragon/configulator"
	"github.com/USA-RedDragon/mesh-manager/internal/config"
	"github.com/USA-RedDragon/mesh-manager/internal/db"
	"github.com/USA-RedDragon/mesh-manager/internal/db/models"
	"github.com/USA-RedDragon/mesh-manager/internal/events"
	"github.com/USA-RedDragon/mesh-manager/internal/ifacewatcher"
	"github.com/USA-RedDragon/mesh-manager/internal/metrics"
	"github.com/USA-RedDragon/mesh-manager/internal/server"
	"github.com/USA-RedDragon/mesh-manager/internal/services"
	"github.com/USA-RedDragon/mesh-manager/internal/services/babel"
	"github.com/USA-RedDragon/mesh-manager/internal/services/dnsmasq"
	"github.com/USA-RedDragon/mesh-manager/internal/services/lqm"
	"github.com/USA-RedDragon/mesh-manager/internal/services/meshlink"
	"github.com/USA-RedDragon/mesh-manager/internal/services/olsr"
	"github.com/USA-RedDragon/mesh-manager/internal/wireguard"
	"github.com/spf13/cobra"
	"github.com/ztrue/shutdown"
	"golang.org/x/sync/errgroup"
)

func newServerCommand(version, commit string) *cobra.Command {
	return &cobra.Command{
		Use:     "server",
		Version: fmt.Sprintf("%s - %s", version, commit),
		Short:   "Start the daemon server",
		Annotations: map[string]string{
			"version": version,
			"commit":  commit,
		},
		RunE:              runServer,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}
}

func runServer(cmd *cobra.Command, _ []string) error {
	err := runRoot(cmd, nil)
	if err != nil {
		slog.Error("Encountered an error.", "error", err.Error())
	}

	ctx := cmd.Context()

	c, err := configulator.FromContext[config.Config](ctx)
	if err != nil {
		return fmt.Errorf("failed to get config from context")
	}

	config, err := c.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Start the server
	slog.Info("Starting server")

	serviceRegistry := services.NewServiceRegistry()
	if config.OLSR {
		serviceRegistry.Register(services.OLSRServiceName, olsr.NewService(config))
	}
	if config.Babel.Enabled {
		serviceRegistry.Register(services.BabelServiceName, babel.NewService(config))
		serviceRegistry.Register(services.MeshLinkServiceName, meshlink.NewService(config))
	}
	serviceRegistry.Register(services.DNSMasqServiceName, dnsmasq.NewService(config))
	serviceRegistry.Register(services.LQMServiceName, lqm.NewService(config))

	go serviceRegistry.StartAll()

	// Start the metrics server
	go metrics.CreateMetricsServer(config, cmd.Root().Version)
	slog.Info("Metrics server started")

	db, err := db.MakeDB(config)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	slog.Info("Database connection established")

	// Clear active status from all tunnels in the db
	err = models.ClearActiveFromAllTunnels(db)
	if err != nil {
		return err
	}
	slog.Info("Cleared active status from all tunnels in the database")

	// Start the wireguard manager
	wireguardManager, err := wireguard.NewManager(db)
	if err != nil {
		return err
	}
	slog.Info("Wireguard manager initialized")

	err = wireguardManager.Run()
	if err != nil {
		return err
	}
	slog.Info("Wireguard manager started")

	if config.OLSR {
		// Run the OLSR metrics watcher
		go metrics.OLSRWatcher(db)
		slog.Info("OLSR metrics watcher started")
	}

	// Initialize the websocket event bus
	eventBus := events.NewEventBus()
	slog.Info("Event bus initialized")

	// Start the interface watcher
	ifWatcher, err := ifacewatcher.NewWatcher(db, eventBus.GetChannel())
	if err != nil {
		return err
	}
	err = ifWatcher.Watch()
	if err != nil {
		return err
	}
	slog.Info("Interface watcher started")

	// Start the server
	srv := server.NewServer(config, db, ifWatcher.Stats, eventBus.GetChannel(), wireguardManager)
	err = srv.Run(cmd.Root().Version, serviceRegistry)
	if err != nil {
		return err
	}
	slog.Info("Server is running")

	stopChan := make(chan error, 1)
	defer close(stopChan)
	stop := func(sig os.Signal) {
		// Extra newline to clear potential terminal control character
		fmt.Println()
		switch sig {
		case syscall.SIGINT:
			slog.Info("Received SIGINT, shutting down")
		case syscall.SIGKILL:
			slog.Info("Received SIGKILL, shutting down")
		case syscall.SIGTERM:
			slog.Info("Received SIGTERM, shutting down")
		case syscall.SIGQUIT:
			slog.Info("Received SIGQUIT, shutting down")
		}
		errGrp := errgroup.Group{}
		errGrp.SetLimit(1)

		errGrp.Go(func() error {
			slog.Debug("Stopping wireguard manager")
			defer slog.Debug("Wireguard manager stopped")
			return wireguardManager.Stop()
		})

		errGrp.Go(func() error {
			slog.Debug("Stopping server")
			defer slog.Debug("Server stopped")
			return srv.Stop()
		})

		errGrp.Go(func() error {
			slog.Debug("Stopping interface watcher")
			defer slog.Debug("Interface watcher stopped")
			return ifWatcher.Stop()
		})

		errGrp.Go(func() error {
			slog.Debug("Clearing active status from all tunnels")
			defer slog.Debug("Cleared active status from all tunnels")
			return models.ClearActiveFromAllTunnels(db)
		})

		errGrp.Go(func() error {
			slog.Debug("Stopping event bus")
			defer slog.Debug("Event bus stopped")
			eventBus.Close()
			return nil
		})

		errGrp.Go(func() error {
			slog.Debug("Stopping service registry")
			defer slog.Debug("Service registry stopped")
			return serviceRegistry.StopAll()
		})

		slog.Debug("Waiting for all errgroups to stop")
		stopChan <- errGrp.Wait()
		slog.Debug("All errgroups stopped")
	}
	shutdown.AddWithParam(stop)
	slog.Info("Signal hooks added")
	shutdown.Listen(syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)
	slog.Info("Signal listener returned")

	return <-stopChan
}
