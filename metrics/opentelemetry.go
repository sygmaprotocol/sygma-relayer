// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

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
func NewTelemetry(meter metric.Meter, env, relayerID string) *Telemetry {
	coreTelemetry := opentelemetry.NewOpenTelemetry(meter)
	metrics := NewMetrics(meter, env, relayerID)

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
