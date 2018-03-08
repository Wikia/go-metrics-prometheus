package prometheusmetrics

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rcrowley/go-metrics"
)

type MetricConverter func(name string, metric interface{}) (float64, error)
type Normalizer func(name string) string

// PrometheusConfig provides a container with config parameters for the
// Prometheus Exporter

type PrometheusConfig struct {
	Namespace     string
	registry      metrics.Registry // Registry to be exported
	Subsystem     string
	promRegistry  prometheus.Registerer //Prometheus registry
	FlushInterval time.Duration         //interval to update prom metrics
	gauges        map[string]prometheus.Gauge
	converter     MetricConverter
	keyNormalizer Normalizer
}

type optSetter func(c *PrometheusConfig) error

func Converter(converter MetricConverter) optSetter {
	return func(c *PrometheusConfig) error {
		c.converter = converter
		return nil
	}
}

func KeyNormalizer(normalizer Normalizer) optSetter {
	return func(c *PrometheusConfig) error {
		c.keyNormalizer = normalizer
		return nil
	}
}

func FlushRate(duration time.Duration) optSetter {
	return func(c *PrometheusConfig) error {
		c.FlushInterval = duration
		return nil
	}
}

func DefaultMetricConverter(name string, i interface{}) (float64, error) {
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

	return 0.0, fmt.Errorf("metric '%s' has unknown type: %s", name, reflect.TypeOf(i))
}

func DefaultKeyNormalizer(key string) string {
	key = strings.Replace(key, " ", "_", -1)
	key = strings.Replace(key, ".", "_", -1)
	key = strings.Replace(key, "-", "_", -1)
	key = strings.Replace(key, "=", "_", -1)

	return key
}

func LowerCaseKeyNormalizer(key string) string { return strings.ToLower(DefaultKeyNormalizer(key)) }

// NewPrometheusProvider returns a Provider that produces Prometheus metrics.
// Namespace and Subsystem are applied to all produced metrics.
func NewPrometheusProvider(r metrics.Registry, namespace string, subsystem string, promRegistry prometheus.Registerer, setters ...optSetter) (*PrometheusConfig, error) {
	conf := &PrometheusConfig{
		Namespace:     namespace,
		Subsystem:     subsystem,
		registry:      r,
		promRegistry:  promRegistry,
		FlushInterval: 15 * time.Second,
		gauges:        make(map[string]prometheus.Gauge),
		converter:     DefaultMetricConverter,
		keyNormalizer: DefaultKeyNormalizer,
	}

	for _, s := range setters {
		if err := s(conf); err != nil {
			return nil, err
		}
	}

	return conf, nil
}

func (c *PrometheusConfig) gaugeFromNameAndValue(name string, val float64) {
	key := fmt.Sprintf("%s_%s_%s", c.Namespace, c.Subsystem, name)
	g, ok := c.gauges[key]
	if !ok {
		g = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: c.keyNormalizer(c.Namespace),
			Subsystem: c.keyNormalizer(c.Subsystem),
			Name:      c.keyNormalizer(name),
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
	c.registry.Each(func(name string, i interface{}) {
		value, err := c.converter(name, i)
		if err == nil {
			c.gaugeFromNameAndValue(name, value)
		}
	})
	return nil
}
