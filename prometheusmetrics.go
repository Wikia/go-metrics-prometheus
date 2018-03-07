package prometheusmetrics

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rcrowley/go-metrics"
)

type MetricConverter func(metric interface{}) (float64, error)

// PrometheusConfig provides a container with config parameters for the
// Prometheus Exporter

type PrometheusConfig struct {
	namespace     string
	Registry      metrics.Registry // Registry to be exported
	subsystem     string
	promRegistry  prometheus.Registerer //Prometheus registry
	FlushInterval time.Duration         //interval to update prom metrics
	gauges        map[string]prometheus.Gauge
	converter     MetricConverter
}

func DefaultMetricConverter(i interface{}) (float64, error) {
	switch metric := i.(type) {
	case metrics.Counter:
		return float64(metric.Count()), nil
	case metrics.Gauge:
		return float64(metric.Value()), nil
	case metrics.GaugeFloat64:
		return float64(metric.Value()), nil
	case metrics.Histogram:
		samples := metric.Snapshot().Sample().Values()
		if len(samples) > 0 {
			lastSample := samples[len(samples)-1]
			return float64(lastSample), nil
		}
	case metrics.Meter:
		lastSample := metric.Snapshot().Rate1()
		return float64(lastSample), nil
	case metrics.Timer:
		lastSample := metric.Snapshot().Rate1()
		return float64(lastSample), nil
	}

	return 0.0, fmt.Errorf("unknown type to convert: %s", reflect.TypeOf(i))
}

// NewPrometheusProvider returns a Provider that produces Prometheus metrics.
// Namespace and subsystem are applied to all produced metrics.
func NewPrometheusProvider(r metrics.Registry, namespace string, subsystem string, promRegistry prometheus.Registerer, FlushInterval time.Duration) *PrometheusConfig {
	return &PrometheusConfig{
		namespace:     namespace,
		subsystem:     subsystem,
		Registry:      r,
		promRegistry:  promRegistry,
		FlushInterval: FlushInterval,
		gauges:        make(map[string]prometheus.Gauge),
		converter:     DefaultMetricConverter,
	}
}

func (c *PrometheusConfig) SetMetricConverter(converter MetricConverter) {
	c.converter = converter
}

func (c *PrometheusConfig) flattenKey(key string) string {
	key = strings.Replace(key, " ", "_", -1)
	key = strings.Replace(key, ".", "_", -1)
	key = strings.Replace(key, "-", "_", -1)
	key = strings.Replace(key, "=", "_", -1)
	return key
}

func (c *PrometheusConfig) gaugeFromNameAndValue(name string, val float64) {
	key := fmt.Sprintf("%s_%s_%s", c.namespace, c.subsystem, name)
	g, ok := c.gauges[key]
	if !ok {
		g = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.flattenKey(c.namespace),
			Subsystem: c.flattenKey(c.subsystem),
			Name:      c.flattenKey(name),
			Help:      name,
		})
		c.promRegistry.MustRegister(g)
		c.gauges[key] = g
	}
	g.Set(val)
}
func (c *PrometheusConfig) UpdatePrometheusMetrics() {
	for _ = range time.Tick(c.FlushInterval) {
		c.UpdatePrometheusMetricsOnce()
	}
}

func (c *PrometheusConfig) UpdatePrometheusMetricsOnce() error {
	c.Registry.Each(func(name string, i interface{}) {
		value, err := c.converter(i)
		if err == nil {
			c.gaugeFromNameAndValue(name, value)
		}
	})
	return nil
}
