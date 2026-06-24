//go:build windows

package collector

import "testing"

func TestFileTimeSeconds(t *testing.T) {
	got := fileTimeSeconds(fileTime{lowDateTime: 10000000})
	if got != 1 {
		t.Fatalf("expected 1 second, got %v", got)
	}
}

func TestWindowsCPUMetrics(t *testing.T) {
	metrics := windowsCPUMetrics(windowsCPUSample{idle: 1, system: 2, user: 3, total: 6})
	if len(metrics) != 3 {
		t.Fatalf("expected 3 metrics, got %d", len(metrics))
	}
}
