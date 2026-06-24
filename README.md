# Go System Monitor CLI

A lightweight Linux system monitoring CLI written in Go. It collects CPU,
memory, and disk metrics concurrently and prints Prometheus-compatible text
that can be scraped, redirected, or plugged into existing observability
pipelines.

## Features

- Concurrent metric collection with goroutines and channels.
- CPU counters from `/proc/stat`, including per-mode seconds and used ratio.
- Memory gauges from `/proc/meminfo`, including total, available, free, swap,
  and used ratio.
- Disk gauges from `statfs`, including size, free, available, and used ratio.
- Single static binary build path for Linux deployment.
- One-shot output by default, with optional watch mode.

## Requirements

- Go 1.25 or newer.
- Linux, Windows, or WSL.
- Linux provides CPU mode counters from `/proc/stat`.
- Windows provides logical CPU cores, physical memory, page file, and disk
  metrics through native Windows APIs.

## Run

```bash
go run ./cmd/gosysmon
```

On Windows PowerShell:

```powershell
go run .\cmd\gosysmon
```

Collect disk metrics for multiple mount points:

```bash
go run ./cmd/gosysmon --path / --path /var
```

On Windows, pass drive paths:

```powershell
go run .\cmd\gosysmon --path C:\ --path W:\
```

Refresh continuously:

```bash
go run ./cmd/gosysmon --watch --interval 10s
```

Example output:

```text
# HELP gosysmon_cpu_seconds_total Seconds the CPUs spent in each mode since boot.
# TYPE gosysmon_cpu_seconds_total counter
gosysmon_cpu_seconds_total{mode="user"} 1234.56
# HELP gosysmon_memory_available_bytes Estimated physical memory available for new workloads in bytes.
# TYPE gosysmon_memory_available_bytes gauge
gosysmon_memory_available_bytes 6733971456
```

## Build

Build a statically linked Linux binary:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/gosysmon ./cmd/gosysmon
```

On Windows PowerShell:

```powershell
$env:CGO_ENABLED='0'; $env:GOOS='linux'; $env:GOARCH='amd64'
go build -trimpath -ldflags="-s -w" -o bin/gosysmon ./cmd/gosysmon
```

## Verify

```bash
go test ./...
go vet ./...
```
