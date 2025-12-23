package metrics

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/USA-RedDragon/mesh-manager/internal/config"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

//nolint:gochecknoglobals
var (
	MeshRF = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "node_details_meshrf",
		Help: "Mesh RF Enabled",
	})
	Info = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_aredn_info",
		Help: "Node Info",
	}, []string{
		"board_id",
		"description",
		"firmware_version",
		"gridsquare",
		"lat",
		"lon",
		"model",
		"node",
		"tactical",
	})
)

func CreateMetricsServer(config *config.Config, version string) {
	if config.Metrics.Enabled {
		// We don't use RF, so we set it to 0
		MeshRF.Set(0)
		Info.WithLabelValues(
			"0x0000",
			"Cloud Tunnel",
			version,
			config.Gridsquare,
			fmt.Sprintf("%f", config.Latitude),
			fmt.Sprintf("%f", config.Longitude),
			"Virtual",
			config.ServerName,
			"",
		).Set(1)
		http.Handle("/metrics", promhttp.Handler())
		server := &http.Server{
			Addr:              fmt.Sprintf(":%d", config.Metrics.Port),
			ReadHeaderTimeout: 5 * time.Second,
		}
		err := server.ListenAndServe()
		if err != nil {
			slog.Error("Error starting metrics server", "error", err)
		}
	}
}
