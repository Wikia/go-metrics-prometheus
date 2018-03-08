package prometheusmetrics

import (
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

func TestPrometheusRegistration(t *testing.T) {
	defaultRegistry := prometheus.DefaultRegisterer
	pClient, _ := NewPrometheusProvider(metrics.DefaultRegistry, "test", "subsys", defaultRegistry, FlushRate(1*time.Second))
	assert.Equal(t, pClient.promRegistry, defaultRegistry, "registries are different")
}

func TestUpdatePrometheusMetricsOnce(t *testing.T) {
	prometheusRegistry := prometheus.NewRegistry()
	metricsRegistry := metrics.NewRegistry()
	pClient, _ := NewPrometheusProvider(metricsRegistry, "test", "subsys", prometheusRegistry, FlushRate(1*time.Second))
	metricsRegistry.Register("counter", metrics.NewCounter())
	pClient.UpdatePrometheusMetricsOnce()
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "test",
		Subsystem: "subsys",
		Name:      "counter",
		Help:      "counter",
	})
	err := prometheusRegistry.Register(gauge)
	assert.Error(t, err, "could not register gauge in the prometheus registry")
}

func TestUpdatePrometheusMetrics(t *testing.T) {
	prometheusRegistry := prometheus.NewRegistry()
	metricsRegistry := metrics.NewRegistry()
	pClient, _ := NewPrometheusProvider(metricsRegistry, "test", "subsys", prometheusRegistry, FlushRate(1*time.Second))
	metricsRegistry.Register("counter", metrics.NewCounter())
	go pClient.UpdatePrometheusMetrics()
	time.Sleep(2 * time.Second)
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "test",
		Subsystem: "subsys",
		Name:      "counter",
		Help:      "counter",
	})
	err := prometheusRegistry.Register(gauge)
	assert.Error(t, err, "could not register gauge in the prometheus registry")
}

func TestPrometheusCounterGetUpdated(t *testing.T) {
	prometheusRegistry := prometheus.NewRegistry()
	metricsRegistry := metrics.NewRegistry()
	pClient, _ := NewPrometheusProvider(metricsRegistry, "test", "subsys", prometheusRegistry, FlushRate(1*time.Second))
	cntr := metrics.NewCounter()
	metricsRegistry.Register("counter", cntr)
	cntr.Inc(2)
	go pClient.UpdatePrometheusMetrics()
	cntr.Inc(13)
	time.Sleep(5 * time.Second)
	metrics, _ := prometheusRegistry.Gather()
	serialized := fmt.Sprint(metrics[0])
	expected := fmt.Sprintf("name:\"test_subsys_counter\" help:\"counter\" type:GAUGE metric:<gauge:<value:%d > > ", cntr.Count())
	assert.Equal(t, expected, serialized, "metrics differ")
}

func TestPrometheusCounterGetUpdatedWithCustomConverter(t *testing.T) {
	prometheusRegistry := prometheus.NewRegistry()
	metricsRegistry := metrics.NewRegistry()
	converter := func(_ string, i interface{}) (float64, error) { return 12345, nil }
	pClient, _ := NewPrometheusProvider(metricsRegistry, "test", "subsys", prometheusRegistry, FlushRate(1*time.Second), Converter(converter))
	cntr := metrics.NewCounter()
	metricsRegistry.Register("counter", cntr)
	cntr.Inc(2)
	go pClient.UpdatePrometheusMetrics()
	cntr.Inc(13)
	time.Sleep(5 * time.Second)
	metrics, _ := prometheusRegistry.Gather()
	serialized := fmt.Sprint(metrics[0])
	expected := fmt.Sprintf("name:\"test_subsys_counter\" help:\"counter\" type:GAUGE metric:<gauge:<value:%d > > ", 12345)
	assert.Equal(t, expected, serialized, "metrics differ")
}

func TestPrometheusLowercaseNormalizer(t *testing.T) {
	prometheusRegistry := prometheus.NewRegistry()
	metricsRegistry := metrics.NewRegistry()
	pClient, _ := NewPrometheusProvider(metricsRegistry, "test", "subsys", prometheusRegistry, FlushRate(1*time.Second), KeyNormalizer(LowerCaseKeyNormalizer))
	cntr := metrics.NewCounter()
	metricsRegistry.Register("Counter", cntr)
	cntr.Inc(2)
	go pClient.UpdatePrometheusMetrics()
	cntr.Inc(13)
	time.Sleep(5 * time.Second)
	metrics, _ := prometheusRegistry.Gather()
	serialized := fmt.Sprint(metrics[0])
	expected := fmt.Sprintf("name:\"test_subsys_counter\" help:\"Counter\" type:GAUGE metric:<gauge:<value:%d > > ", cntr.Count())
	assert.Equal(t, expected, serialized, "metrics differ")
}

func TestPrometheusGaugeGetUpdated(t *testing.T) {
	prometheusRegistry := prometheus.NewRegistry()
	metricsRegistry := metrics.NewRegistry()
	pClient, _ := NewPrometheusProvider(metricsRegistry, "test", "subsys", prometheusRegistry, FlushRate(1*time.Second))
	gm := metrics.NewGauge()
	metricsRegistry.Register("gauge", gm)
	gm.Update(2)
	go pClient.UpdatePrometheusMetrics()
	gm.Update(13)
	time.Sleep(5 * time.Second)
	metrics, _ := prometheusRegistry.Gather()
	assert.Equal(t, 1, len(metrics), "prometheus was unable to register the metric")
	serialized := fmt.Sprint(metrics[0])
	expected := fmt.Sprintf("name:\"test_subsys_gauge\" help:\"gauge\" type:GAUGE metric:<gauge:<value:%d > > ", gm.Value())
	assert.Equal(t, expected, serialized, "metrics differ")
}

func TestPrometheusMeterGetUpdated(t *testing.T) {
	prometheusRegistry := prometheus.NewRegistry()
	metricsRegistry := metrics.NewRegistry()
	pClient, _ := NewPrometheusProvider(metricsRegistry, "test", "subsys", prometheusRegistry, FlushRate(1*time.Second))
	gm := metrics.NewMeter()
	metricsRegistry.Register("meter", gm)
	gm.Mark(2)
	go pClient.UpdatePrometheusMetrics()
	gm.Mark(13)
	time.Sleep(5 * time.Second)
	metrics, _ := prometheusRegistry.Gather()
	assert.Equal(t, 1, len(metrics), "prometheus was unable to register the metric")
	serialized := fmt.Sprint(metrics[0])
	expected := fmt.Sprintf("name:\"test_subsys_meter\" help:\"meter\" type:GAUGE metric:<gauge:<value:%g > > ", gm.Rate1())
	assert.Equal(t, expected, serialized, "metrics differ")
}
