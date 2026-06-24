//go:build windows

package collector

import (
	"fmt"
	"time"
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
	first, err := windowsCPUSnapshot()
	if err != nil {
		return nil, err
	}
	time.Sleep(500 * time.Millisecond)
	second, err := windowsCPUSnapshot()
	if err != nil {
		return nil, err
	}

	metrics := windowsCPUMetrics(second)
	totalDelta := second.total - first.total
	idleDelta := second.idle - first.idle
	if totalDelta > 0 {
		metrics = append(metrics, Metric{
			Name:  "gosysmon_cpu_used_ratio",
			Help:  "Current non-idle CPU time ratio sampled over a short interval.",
			Type:  Gauge,
			Value: (totalDelta - idleDelta) / totalDelta,
		})
	}

	return metrics, nil
}

type windowsCPUSample struct {
	idle   float64
	system float64
	user   float64
	total  float64
}

func windowsCPUSnapshot() (windowsCPUSample, error) {
	var idle, kernel, user fileTime
	ret, _, err := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&idle)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if ret == 0 {
		return windowsCPUSample{}, fmt.Errorf("GetSystemTimes: %w", err)
	}

	idleSeconds := fileTimeSeconds(idle)
	kernelSeconds := fileTimeSeconds(kernel)
	userSeconds := fileTimeSeconds(user)
	systemSeconds := kernelSeconds - idleSeconds
	if systemSeconds < 0 {
		systemSeconds = 0
	}

	return windowsCPUSample{
		idle:   idleSeconds,
		system: systemSeconds,
		user:   userSeconds,
		total:  idleSeconds + systemSeconds + userSeconds,
	}, nil
}

func windowsCPUMetrics(sample windowsCPUSample) []Metric {
	return []Metric{
		{
			Name:   "gosysmon_cpu_seconds_total",
			Help:   "Seconds the CPUs spent in each mode since boot.",
			Type:   Counter,
			Labels: map[string]string{"mode": "idle"},
			Value:  sample.idle,
		},
		{
			Name:   "gosysmon_cpu_seconds_total",
			Help:   "Seconds the CPUs spent in each mode since boot.",
			Type:   Counter,
			Labels: map[string]string{"mode": "system"},
			Value:  sample.system,
		},
		{
			Name:   "gosysmon_cpu_seconds_total",
			Help:   "Seconds the CPUs spent in each mode since boot.",
			Type:   Counter,
			Labels: map[string]string{"mode": "user"},
			Value:  sample.user,
		},
	}
}

func fileTimeSeconds(value fileTime) float64 {
	ticks := uint64(value.highDateTime)<<32 | uint64(value.lowDateTime)
	return float64(ticks) / 10000000
}
