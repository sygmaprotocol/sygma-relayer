package metrics

import (
	"context"

	"github.com/ChainSafe/chainbridge-core/opentelemetry"
	"go.opentelemetry.io/otel/metric"
)

type Metrics struct {
	opentelemetry.ChainbridgeMetrics
	DepositErrorRate  metric.Int64Counter
	TotalRelayers     metric.Int64GaugeObserver
	AvailableRelayers metric.Int64GaugeObserver
	ExecutionLatency  metric.Int64Histogram

	TotalRelayerCount     *int64
	AvailableRelayerCount *int64
}

// NewMetrics creates an instance of metrics
func NewMetrics(meter metric.Meter) *Metrics {
	totalRelayerCount := new(int64)
	availableRelayerCount := new(int64)
	return &Metrics{
		ChainbridgeMetrics: *opentelemetry.NewChainbridgeMetrics(meter),
		TotalRelayers: metric.Must(meter).NewInt64GaugeObserver(
			"sygma.TotalRelayers",
			func(ctx context.Context, result metric.Int64ObserverResult) {
				result.Observe(*totalRelayerCount)
			},
			metric.WithDescription("Total number of relayers currently in the subset"),
		),
		AvailableRelayers: metric.Must(meter).NewInt64GaugeObserver(
			"sygma.AvailableRelayers",
			func(ctx context.Context, result metric.Int64ObserverResult) {
				result.Observe(*availableRelayerCount)
			},
			metric.WithDescription("Available number of relayers currently in the subset"),
		),
		TotalRelayerCount:     totalRelayerCount,
		AvailableRelayerCount: availableRelayerCount,
	}
}
