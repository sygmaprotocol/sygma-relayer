// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package metrics

import (
	"context"
	"github.com/ChainSafe/chainbridge-core/opentelemetry"
	"github.com/libp2p/go-libp2p/core/peer"
	api "go.opentelemetry.io/otel/metric"
)

type SygmaMetrics struct {
	opentelemetry.RelayerMetrics

	meter api.Meter

	DepositErrorRate  api.Int64Counter
	TotalRelayers     api.Int64ObservableGauge
	AvailableRelayers api.Int64ObservableGauge
	ExecutionLatency  api.Int64Histogram

	TotalRelayerCount     *int64
	AvailableRelayerCount *int64
}

// NewMetrics creates an instance of metrics
func NewSygmaMetrics(meter api.Meter, env, relayerID string) (*SygmaMetrics, error) {
	relayerMetrics, err := opentelemetry.NewRelayerMetrics(meter, env, relayerID)
	if err != nil {
		return nil, err
	}

	totalRelayerGauge := new(int64)
	availableRelayerGauge := new(int64)
	totalRelayersCount, err := meter.Int64ObservableGauge(
		"relayer.TotalRelayers",
		api.WithInt64Callback(func(context context.Context, result api.Int64Observer) error {
			result.Observe(*availableRelayerGauge, relayerMetrics.Opts)
			return nil
		}),
		api.WithDescription("Total number of relayers currently in the subset"),
	)
	if err != nil {
		return nil, err
	}
	availableRelayersCount, err := meter.Int64ObservableGauge(
		"relayer.AvalableRelayers",
		api.WithInt64Callback(func(context context.Context, result api.Int64Observer) error {
			result.Observe(*availableRelayerGauge, relayerMetrics.Opts)
			return nil
		}),
		api.WithDescription("Available number of relayers currently in the subset"),
	)
	if err != nil {
		return nil, err
	}

	return &SygmaMetrics{
		RelayerMetrics:        *relayerMetrics,
		TotalRelayers:         totalRelayersCount,
		AvailableRelayers:     availableRelayersCount,
		TotalRelayerCount:     totalRelayerGauge,
		AvailableRelayerCount: availableRelayerGauge,
	}, nil
}

func (t *SygmaMetrics) TrackRelayerStatus(unavailable peer.IDSlice, all peer.IDSlice) {
	*t.TotalRelayerCount = int64(len(all))
	*t.AvailableRelayerCount = int64(len(all) - len(unavailable))
}
