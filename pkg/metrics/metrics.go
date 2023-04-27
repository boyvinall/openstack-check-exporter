package metrics

import (
	"sync"
	"time"

	"github.com/boyvinall/openstack-check-exporter/pkg/checker"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	lock   sync.Mutex
	latest []*checker.CheckResult
}

func New() *Metrics {
	m := &Metrics{}
	prometheus.Register(m)
	return m
}

func (m *Metrics) Update(results []*checker.CheckResult) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.latest = results
}

func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(m, ch)
}

// Collect implements the prometheus.Collector interface
func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for _, r := range m.latest {
		up := 1.0
		if r.Error != nil {
			up = 0.0
		}

		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				"openstack_check_healthy",
				"OpenStack Monitoring Check",
				[]string{
					"name",
					"cloud",
				},
				nil,
			),
			prometheus.GaugeValue,
			up,
			r.Name,
			r.Cloud,
		)

		ch <- prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				"openstack_check_duration_seconds",
				"How long the check took to run",
				[]string{
					"name",
					"cloud",
				},
				nil,
			),
			prometheus.GaugeValue,
			float64(r.Duration)/float64(time.Second),
			r.Name,
			r.Cloud,
		)
	}
}
