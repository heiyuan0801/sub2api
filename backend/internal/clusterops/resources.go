package clusterops

import (
	"context"
	"math"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
)

// Resources describes the OS-visible system, not a container quota. Process RSS
// is reported separately. Missing measurements must not look like zero usage.
type Resources struct {
	SampledAt        time.Time `json:"sampled_at"`
	Hostname         string    `json:"hostname"`
	OS               string    `json:"os"`
	Arch             string    `json:"arch"`
	CPUCores         int       `json:"cpu_cores"`
	CPUPercent       *float64  `json:"cpu_percent"`
	CPUWindowSeconds *float64  `json:"cpu_window_seconds"`
	Load1            *float64  `json:"load_1"`
	Load5            *float64  `json:"load_5"`
	Load15           *float64  `json:"load_15"`
	MemoryTotal      uint64    `json:"memory_total_bytes"`
	MemoryUsed       uint64    `json:"memory_used_bytes"`
	MemoryAvailable  uint64    `json:"memory_available_bytes"`
	MemoryPercent    *float64  `json:"memory_percent"`
	ProcessRSS       *uint64   `json:"process_rss_bytes"`
	UptimeSeconds    *uint64   `json:"uptime_seconds"`
}

func (m *Manager) collectResources(ctx context.Context) {
	defer close(m.resourcesDone)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	var collector resourceCollector
	for {
		if ctx.Err() != nil {
			return
		}
		probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		sample := collector.collect(probeCtx)
		cancel()
		m.resources.Store(sample)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

type resourceCollector struct {
	previous   *cpu.TimesStat
	previousAt time.Time
}

func (c *resourceCollector) collect(ctx context.Context) *Resources {
	now := time.Now()
	r := &Resources{SampledAt: now, OS: runtime.GOOS, Arch: runtime.GOARCH, CPUCores: runtime.NumCPU()}
	r.Hostname, _ = os.Hostname()
	// Own the baseline instead of cpu.Percent(0), whose shared baseline can be
	// advanced by the existing operations collector in the same process.
	if times, err := cpu.TimesWithContext(ctx, false); err == nil && len(times) > 0 {
		current := times[0]
		if c.previous != nil {
			r.CPUPercent = cpuUtilization(*c.previous, current)
			if r.CPUPercent != nil {
				window := now.Sub(c.previousAt).Seconds()
				r.CPUWindowSeconds = &window
			}
		}
		c.previous, c.previousAt = &current, now
	} else {
		c.previous = nil
	}
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil && vm.Total > 0 && vm.Available <= vm.Total {
		r.MemoryTotal, r.MemoryAvailable = vm.Total, vm.Available
		r.MemoryUsed = vm.Total - vm.Available
		percent := 100 * float64(r.MemoryUsed) / float64(vm.Total)
		r.MemoryPercent = &percent
	}
	if avg, err := load.AvgWithContext(ctx); err == nil {
		r.Load1, r.Load5, r.Load15 = finiteMetric(avg.Load1), finiteMetric(avg.Load5), finiteMetric(avg.Load15)
	}
	if uptime, err := host.UptimeWithContext(ctx); err == nil {
		r.UptimeSeconds = &uptime
	}
	if p, err := process.NewProcessWithContext(ctx, int32(os.Getpid())); err == nil {
		if info, err := p.MemoryInfoWithContext(ctx); err == nil {
			r.ProcessRSS = &info.RSS
		}
	}
	return r
}

func finiteMetric(v float64) *float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return nil
	}
	return &v
}

func cpuUtilization(previous, current cpu.TimesStat) *float64 {
	// Guest time is already included in user/nice; counting it twice inflates CPU.
	total := func(s cpu.TimesStat) float64 {
		return s.User + s.System + s.Nice + s.Idle + s.Iowait + s.Irq + s.Softirq + s.Steal
	}
	delta := total(current) - total(previous)
	idle := current.Idle + current.Iowait - previous.Idle - previous.Iowait
	if delta <= 0 || idle < 0 || idle > delta {
		return nil
	}
	return finiteMetric(100 * (delta - idle) / delta)
}
