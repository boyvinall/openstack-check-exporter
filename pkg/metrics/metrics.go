// Package metrics implements a prometheus.Collector that exposes metrics about the checks that were run
package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
)

// Metrics implements a prometheus.Collector that exposes metrics about the checks that were run
type Metrics struct {
	healthy    *prometheus.GaugeVec
	duration   *prometheus.GaugeVec
	lastUpdate *prometheus.GaugeVec
}

// New returns a new Metrics instance
func New() *Metrics {
	m := &Metrics{
		healthy: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "openstack_check_healthy",
				Help: "OpenStack Monitoring Check",
			},
			[]string{
				"name",
				"cloud",
			}),
		duration: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "openstack_check_duration_seconds",
				Help: "How long the check took to run",
			},
			[]string{
				"name",
				"cloud",
			}),
		lastUpdate: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "openstack_check_last_update_time_seconds",
				Help: "Number of seconds since epoch when check was last updated",
			},
			[]string{
				"name",
				"cloud",
			}),
	}

	prometheus.MustRegister(m.healthy)
	prometheus.MustRegister(m.duration)
	prometheus.MustRegister(m.lastUpdate)
	return m
}

// Update updates the metrics with the latest check results
func (m *Metrics) Update(r checker.CheckResult) {
	up := 1
	if r.Error != nil {
		up = 0
	}
	duration := float64(r.Duration) / float64(time.Second)
	end := r.Start.Add(r.Duration).UTC().Unix()

	m.healthy.WithLabelValues(r.Name, r.Cloud).Set(float64(up))
	m.duration.WithLabelValues(r.Name, r.Cloud).Set(duration)
	m.lastUpdate.WithLabelValues(r.Name, r.Cloud).Set(float64(end))
}
