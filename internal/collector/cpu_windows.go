//go:build windows

package collector

import (
	"fmt"
	"unsafe"
)

var (
	procGetSystemTimes = kernel32.NewProc("GetSystemTimes")
)

type fileTime struct {
	lowDateTime  uint32
	highDateTime uint32
}

func platformCPUMetrics() ([]Metric, error) {
	var idle, kernel, user fileTime
	ret, _, err := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if ret == 0 {
		return nil, fmt.Errorf("GetSystemTimes: %w", err)
	}

	idleSeconds := fileTimeSeconds(idle)
	kernelSeconds := fileTimeSeconds(kernel)
	userSeconds := fileTimeSeconds(user)
	systemSeconds := kernelSeconds - idleSeconds
	if systemSeconds < 0 {
		systemSeconds = 0
	}

	metrics := []Metric{
		{
			Name:   "gosysmon_cpu_seconds_total",
			Help:   "Seconds the CPUs spent in each mode since boot.",
			Type:   Counter,
			Labels: map[string]string{"mode": "idle"},
			Value:  idleSeconds,
		},
		{
			Name:   "gosysmon_cpu_seconds_total",
			Help:   "Seconds the CPUs spent in each mode since boot.",
			Type:   Counter,
			Labels: map[string]string{"mode": "system"},
			Value:  systemSeconds,
		},
		{
			Name:   "gosysmon_cpu_seconds_total",
			Help:   "Seconds the CPUs spent in each mode since boot.",
			Type:   Counter,
			Labels: map[string]string{"mode": "user"},
			Value:  userSeconds,
		},
	}

	total := idleSeconds + systemSeconds + userSeconds
	if total > 0 {
		metrics = append(metrics, Metric{
			Name:  "gosysmon_cpu_used_ratio",
			Help:  "Approximate non-idle CPU time ratio since boot.",
			Type:  Gauge,
			Value: (systemSeconds + userSeconds) / total,
		})
	}

	return metrics, nil
}

func fileTimeSeconds(value fileTime) float64 {
	ticks := uint64(value.highDateTime)<<32 | uint64(value.lowDateTime)
	return float64(ticks) / 10000000
}
