# Metrics
Metrics are implemented via the OpenTelemetry [stack](https://opentelemetry.io/) and exported to the Opentelemetry [collector](https://opentelemetry.io/docs/collector/) which then be configured to export to supported metrics tools like Datadog.

## Exported metrics
The following metrics are exported:
```
chainbridge.DepositEventCount (counter) - count of indexed deposits
chainbridge.ExecutionErrorCount (counter) - count of executions that failed
chainbridge.ExecutionLatencyPerRoute (histogram) - latency between indexing event and executing it per route
chainbridge.ExecutionLatency (histogram) - latency between indexing event and executing it across all routes
sygma.TotalRelayers (gauge) - number of relayers currently in the subset for MPC
sygma.availableRelayers (gauge) - number of currently available relayers from the subset

```

## Env variables
- SYG_RELAYER_OPENTELEMETERYCOLLECTORURL - url of the opentelemetry collector application that collects metrics
