package metrics

import (
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/pkg/errors"
)

// Collector describes an object that collects metrics
type Collector interface {
	Timing(name string, tags ...string) (end func(moretags ...string))
	Increment(name string, tags ...string)
	Gauge(name string, value float64, tags ...string)
}

// DatadogCollector represents a collector that pushes metrics to Datadog
type DatadogCollector struct {
	c *statsd.Client
}

var _ Collector = &DatadogCollector{}

// NewDatadogCollector returns a DatadogCollector using dogstatsd at addr, using namespace and adding tags to all metrics.
func NewDatadogCollector(namespace, addr string, tags []string) (*DatadogCollector, error) {
	c, err := statsd.New(addr)
	if err != nil {
		return nil, errors.Wrap(err, "error creating statsd client")
	}
	c.Namespace = namespace
	c.Tags = tags
	return &DatadogCollector{
		c: c,
	}, nil
}

// Timing marks the start of a timed operation and returns a function that should be called when the timed operation is finished
// Tags is zero or more strings of the form "[TAG]:[VALUE]". Optional additional tags can be provided when end() is called.
func (dc *DatadogCollector) Timing(name string, tags ...string) (end func(moretags ...string)) {
	start := time.Now().UTC()
	return func(moretags ...string) {
		since := time.Since(start)
		alltags := append(tags, moretags...)
		dc.c.Timing(name, since, alltags, 1)
		dc.c.Histogram(name+"_seconds", since.Seconds(), alltags, 1)
	}
}

// Increment increments a counter.
// Tags is zero or more strings of the form "[TAG]:[VALUE]"
func (dc *DatadogCollector) Increment(name string, tags ...string) {
	dc.c.Incr(name, tags, 1)
}

// Gauge sets the value of a gauge.
// Tags is zero or more strings of the form "[TAG]:[VALUE]"
func (dc *DatadogCollector) Gauge(name string, value float64, tags ...string) {
	dc.c.Gauge(name, value, tags, 1)
}
