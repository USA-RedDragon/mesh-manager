package metrics

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/USA-RedDragon/mesh-manager/internal/db/models"
	"github.com/USA-RedDragon/mesh-manager/internal/server/api/apimodels"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
)

//nolint:gochecknoglobals
var (
	olsrLinkLabels = []string{"device", "local_ip", "remote_ip"}

	OLSRLinkAsymmetryTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_asymmetry_time",
		Help: "OLSR Link Asymmetry Time",
	}, olsrLinkLabels)
	OLSRLinkHelloTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_hello_time",
		Help: "OLSR Link Hello Time",
	}, olsrLinkLabels)
	OLSRLinkHysteresis = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_hysteresis",
		Help: "OLSR Link Hysteresis",
	}, olsrLinkLabels)
	OLSRLinkLastHelloTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_last_hello_time",
		Help: "OLSR Link Last Hello Time",
	}, olsrLinkLabels)
	OLSRLinkLinkCost = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_link_cost",
		Help: "OLSR Link Cost",
	}, olsrLinkLabels)
	OLSRLinkLinkQuality = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_link_quality",
		Help: "OLSR Link Quality",
	}, olsrLinkLabels)
	OLSRLinkLossHelloInterval = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_loss_hello_interval",
		Help: "OLSR Link Loss Hello Interval",
	}, olsrLinkLabels)
	OLSRLinkLossMultiplier = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_loss_multiplier",
		Help: "OLSR Link Loss Multiplier",
	}, olsrLinkLabels)
	OLSRLinkLossTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_loss_time",
		Help: "OLSR Link Loss Time",
	}, olsrLinkLabels)
	OLSRLinkLostLinkTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_lost_link_time",
		Help: "OLSR Link Lost Link Time",
	}, olsrLinkLabels)
	OLSRLinkNeighborLinkQuality = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_neighbor_link_quality",
		Help: "OLSR Link Neighbor Link Quality",
	}, olsrLinkLabels)
	OLSRLinkPending = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_pending",
		Help: "OLSR Link Pending",
	}, olsrLinkLabels)
	OLSRLinkSeqno = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_seqno",
		Help: "OLSR Link Seqno",
	}, olsrLinkLabels)
	OLSRLinkSeqnoValid = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_seqno_valid",
		Help: "OLSR Link Seqno Valid",
	}, olsrLinkLabels)
	OLSRLinkSymmetryTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_symmetry_time",
		Help: "OLSR Link Symmetry Time",
	}, olsrLinkLabels)
	OLSRLinkValidityTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_validity_time",
		Help: "OLSR Link Validity Time",
	}, olsrLinkLabels)
	OLSRLinkVTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "node_olsr_link_vtime",
		Help: "OLSR Link VTime",
	}, olsrLinkLabels)
)

func OLSRWatcher(db *gorm.DB) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	for {
		req, err := http.NewRequestWithContext(context.TODO(), http.MethodGet, "http://localhost:9090/links", nil)
		if err != nil {
			slog.Error("OLSRWatcher: Unable to create request", "error", err)
			time.Sleep(1 * time.Second)
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			slog.Error("OLSRWatcher: Unable to get links", "error", err)
			time.Sleep(1 * time.Second)
			continue
		}
		var links apimodels.OlsrdLinks
		err = json.NewDecoder(resp.Body).Decode(&links)
		if err != nil {
			slog.Error("OLSRWatcher: Unable to decode links", "error", err)
			time.Sleep(1 * time.Second)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		foundInterfaces := make([]string, 0, len(links.Links))

		for _, link := range links.Links {
			foundInterfaces = append(foundInterfaces, link.OLSRInterface)

			pending := 0
			if link.Pending {
				pending = 1
			}

			seqnoValid := 0
			if link.SeqnoValid {
				seqnoValid = 1
			}

			OLSRLinkAsymmetryTime.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.AsymmetryTime))
			OLSRLinkHelloTime.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.HelloTime))
			OLSRLinkHysteresis.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.Hysteresis))
			OLSRLinkLastHelloTime.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.LastHelloTime))
			OLSRLinkLinkCost.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.LinkCost))
			OLSRLinkLinkQuality.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.LinkQuality))
			OLSRLinkLossHelloInterval.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.LossHelloInterval))
			OLSRLinkLossMultiplier.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.LossMultiplier))
			OLSRLinkLossTime.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.LossTime))
			OLSRLinkLostLinkTime.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.LostLinkTime))
			OLSRLinkNeighborLinkQuality.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.NeighborLinkQuality))
			OLSRLinkPending.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(pending))
			OLSRLinkSeqno.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.Seqno))
			OLSRLinkSeqnoValid.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(seqnoValid))
			OLSRLinkSymmetryTime.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.SymmetryTime))
			OLSRLinkValidityTime.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.ValidityTime))
			OLSRLinkVTime.WithLabelValues(link.OLSRInterface, link.LocalIP, link.RemoteIP).Set(float64(link.VTime))
		}

		go func() {
			for _, iface := range foundInterfaces {
				if !strings.HasPrefix(iface, "br") {
					tunnel, err := models.FindTunnelByInterface(db, iface)
					if err != nil {
						slog.Error("OLSRWatcher: Unable to find tunnel by interface", "iface", iface, "error", err)
						continue
					}
					tunnel.Active = true
					err = db.Save(&tunnel).Error
					if err != nil {
						slog.Error("OLSRWatcher: Unable to save tunnel", "error", err)
					}
				}
			}
		}()

		time.Sleep(1 * time.Second)
	}
}
