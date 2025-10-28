# Conntrack Configuration

This document describes the configuration options available for the conntrack aggregator.

## Environment Variables

The conntrack aggregator can be configured using environment variables with the `CONNTRACK_` prefix:

| Variable | Default | Description |
|----------|---------|-------------|
| `CONNTRACK_EVENT_CHAN_SIZE` | `524288` | Event channel buffer size (512KB) |
| `CONNTRACK_EVENT_WORKER_COUNT` | `100` | Number of event worker goroutines |
| `CONNTRACK_DESTROY_FLUSH_INTERVAL` | `50ms` | Interval for flushing destroy deltas |
| `CONNTRACK_DESTROY_DELTA_CAP` | `200000` | Maximum destroy delta entries |
| `CONNTRACK_DROPS_WARN_THRESHOLD` | `10000` | Threshold for missed events warning |
| `CONNTRACK_READ_BUFFER_SIZE` | `67108864` | Read buffer size (64MB) |
| `CONNTRACK_WRITE_BUFFER_SIZE` | `67108864` | Write buffer size (64MB) |
| `CONNTRACK_HEALTH_CHECK_INTERVAL` | `5m` | Health check interval |
| `CONNTRACK_GRACEFUL_TIMEOUT` | `30s` | Graceful shutdown timeout |

## Usage Examples

### Basic Configuration

```bash
# Set custom buffer sizes
export CONNTRACK_EVENT_CHAN_SIZE=1048576
export CONNTRACK_EVENT_WORKER_COUNT=200

# Run the exporter
./openvswitch_exporter
```

### High-Throughput Environment

For environments with high conntrack event rates (>1M events/sec):

```bash
export CONNTRACK_EVENT_CHAN_SIZE=1048576        # 1MB buffer
export CONNTRACK_EVENT_WORKER_COUNT=200         # More workers
export CONNTRACK_DESTROY_FLUSH_INTERVAL=25ms    # Faster flushing
export CONNTRACK_DESTROY_DELTA_CAP=500000       # Larger delta cap
export CONNTRACK_READ_BUFFER_SIZE=134217728     # 128MB read buffer
export CONNTRACK_WRITE_BUFFER_SIZE=134217728    # 128MB write buffer
```

### Low-Resource Environment

For environments with limited resources:

```bash
export CONNTRACK_EVENT_CHAN_SIZE=65536          # 64KB buffer
export CONNTRACK_EVENT_WORKER_COUNT=50          # Fewer workers
export CONNTRACK_DESTROY_FLUSH_INTERVAL=100ms   # Slower flushing
export CONNTRACK_DESTROY_DELTA_CAP=50000        # Smaller delta cap
export CONNTRACK_READ_BUFFER_SIZE=16777216      # 16MB read buffer
export CONNTRACK_WRITE_BUFFER_SIZE=16777216     # 16MB write buffer
```

### Development/Testing

For development and testing:

```bash
export CONNTRACK_GRACEFUL_TIMEOUT=5s            # Faster shutdown
export CONNTRACK_HEALTH_CHECK_INTERVAL=1m       # More frequent health checks
```

## Configuration Validation

The configuration system includes validation:

- **Positive values**: All numeric values must be positive
- **Valid durations**: Time values must be valid Go durations
- **Range checks**: Values are checked for reasonable ranges

Invalid values will fall back to defaults with a warning logged.

## Migration from Hardcoded Constants

The following hardcoded constants have been replaced:

| Old Constant | New Environment Variable | Default Value |
|--------------|-------------------------|---------------|
| `eventChanSize = 512 * 1024` | `CONNTRACK_EVENT_CHAN_SIZE` | `524288` |
| `eventWorkerCount = 100` | `CONNTRACK_EVENT_WORKER_COUNT` | `100` |
| `destroyFlushIntvl = 50ms` | `CONNTRACK_DESTROY_FLUSH_INTERVAL` | `50ms` |
| `destroyDeltaCap = 200000` | `CONNTRACK_DESTROY_DELTA_CAP` | `200000` |
| `dropsWarnThreshold = 10000` | `CONNTRACK_DROPS_WARN_THRESHOLD` | `10000` |
| Buffer sizes `64MB` | `CONNTRACK_READ_BUFFER_SIZE` / `WRITE_BUFFER_SIZE` | `67108864` |
| Health check `5m` | `CONNTRACK_HEALTH_CHECK_INTERVAL` | `5m` |
| Graceful timeout `30s` | `CONNTRACK_GRACEFUL_TIMEOUT` | `30s` |

## Performance Impact

Configuration changes can significantly impact performance:

- **Larger buffers**: Better for high-throughput, uses more memory
- **More workers**: Better parallelism, uses more CPU
- **Faster flushing**: Lower latency, more CPU usage
- **Larger delta cap**: Handles bursts better, uses more memory

Choose settings based on your environment's characteristics and requirements.
