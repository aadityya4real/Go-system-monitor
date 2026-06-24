package collector

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
)

type CPUCollector struct {
	StatPath string
}

func (c CPUCollector) Collect(ctx context.Context) ([]Metric, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if c.StatPath == "" {
		metrics, err := platformCPUMetrics()
		if metrics != nil || err != nil {
			return appendLogicalCores(metrics), err
		}
	}

	path := c.StatPath
	if path == "" {
		path = "/proc/stat"
	}

	file, err := os.Open(path)
	if err != nil {
		return []Metric{{
			Name:  "gosysmon_cpu_logical_cores",
			Help:  "Number of logical CPU cores visible to the process.",
			Type:  Gauge,
			Value: float64(runtime.NumCPU()),
		}}, fmt.Errorf("collect cpu from %s: %w", path, err)
	}
	defer file.Close()

	return parseCPUStat(file)
}

func parseCPUStat(r io.Reader) ([]Metric, error) {
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("missing aggregate cpu line")
	}

	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return nil, fmt.Errorf("unexpected aggregate cpu line")
	}

	modes := []string{"user", "nice", "system", "idle", "iowait", "irq", "softirq", "steal", "guest", "guest_nice"}
	metrics := make([]Metric, 0, len(fields))
	var total, idle float64

	for i := 1; i < len(fields) && i <= len(modes); i++ {
		ticks, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return nil, fmt.Errorf("parse cpu %s ticks: %w", modes[i-1], err)
		}
		seconds := ticks / 100
		total += ticks
		if modes[i-1] == "idle" || modes[i-1] == "iowait" {
			idle += ticks
		}
		metrics = append(metrics, Metric{
			Name:   "gosysmon_cpu_seconds_total",
			Help:   "Seconds the CPUs spent in each mode since boot.",
			Type:   Counter,
			Labels: map[string]string{"mode": modes[i-1]},
			Value:  seconds,
		})
	}

	if total > 0 {
		metrics = append(metrics, Metric{
			Name:  "gosysmon_cpu_used_ratio",
			Help:  "Approximate non-idle CPU time ratio since boot.",
			Type:  Gauge,
			Value: (total - idle) / total,
		})
	}

	metrics = append(metrics, Metric{
		Name:  "gosysmon_cpu_logical_cores",
		Help:  "Number of logical CPU cores visible to the process.",
		Type:  Gauge,
		Value: float64(runtime.NumCPU()),
	})

	return metrics, scanner.Err()
}

func logicalCoresMetric() Metric {
	return Metric{
		Name:  "gosysmon_cpu_logical_cores",
		Help:  "Number of logical CPU cores visible to the process.",
		Type:  Gauge,
		Value: float64(runtime.NumCPU()),
	}
}

func appendLogicalCores(metrics []Metric) []Metric {
	return append(metrics, logicalCoresMetric())
}
