package metrics

import (
	"context"

	"github.com/ChainSafe/chainbridge-core/opentelemetry"
	"github.com/libp2p/go-libp2p/core/peer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Telemetry struct {
	opentelemetry.OpenTelemetry
	metrics *Metrics

	meter metric.Meter
}

// NewTelemetry initializes OpenTelementry metrics
func NewTelemetry(meter metric.Meter) *Telemetry {
	coreTelemetry := opentelemetry.NewOpenTelemetry(meter)
	metrics := NewMetrics(meter)

	return &Telemetry{
		OpenTelemetry: *coreTelemetry,
		metrics:       metrics,
		meter:         meter,
	}
}

func (t *Telemetry) TrackRelayerStatus(unavailable peer.IDSlice, all peer.IDSlice) {
	*t.metrics.TotalRelayerCount = int64(len(all))
	*t.metrics.AvailableRelayerCount = int64(len(all) - len(unavailable))
	t.meter.RecordBatch(context.Background(), []attribute.KeyValue{})
}
