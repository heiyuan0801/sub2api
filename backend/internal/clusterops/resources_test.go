package clusterops

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/stretchr/testify/require"
)

func TestCPUUtilizationIntervals(t *testing.T) {
	previous := cpu.TimesStat{User: 100, Idle: 100, Iowait: 10, Guest: 20}
	current := cpu.TimesStat{User: 110, Idle: 120, Iowait: 20, Guest: 25}
	require.InDelta(t, 25, *cpuUtilization(previous, current), 0.001)
	require.Nil(t, cpuUtilization(current, current), "first/empty intervals are unknown")
	require.Nil(t, cpuUtilization(current, previous), "counter reset is unknown")
	require.Equal(t, 0.0, *cpuUtilization(cpu.TimesStat{}, cpu.TimesStat{Idle: 10}))
	require.Equal(t, 100.0, *cpuUtilization(cpu.TimesStat{}, cpu.TimesStat{User: 10}))
	require.Nil(t, finiteMetric(math.NaN()))
	require.Nil(t, finiteMetric(math.Inf(1)))
}

func TestResourceSamplingAndLegacySnapshot(t *testing.T) {
	var legacy Stats
	require.NoError(t, json.Unmarshal([]byte(`{"heap_bytes":12}`), &legacy))
	require.Nil(t, legacy.Resources)
	require.Nil(t, legacy.DatabaseLatencyMS)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var collector resourceCollector
	r := collector.collect(ctx)
	require.NotZero(t, r.SampledAt)
	require.Positive(t, r.CPUCores)
	require.Nil(t, r.CPUPercent, "CPU needs two samples")
	if r.MemoryPercent != nil {
		require.Equal(t, r.MemoryTotal, r.MemoryUsed+r.MemoryAvailable)
		require.GreaterOrEqual(t, *r.MemoryPercent, 0.0)
		require.LessOrEqual(t, *r.MemoryPercent, 100.0)
	}
	_, err := json.Marshal(Stats{Resources: r})
	require.NoError(t, err)
}

func TestResourceLoopStopsWithoutHeartbeat(t *testing.T) {
	m := &Manager{resourcesDone: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	go m.collectResources(ctx)
	select {
	case <-m.resourcesDone:
	case <-time.After(time.Second):
		t.Fatal("collector did not stop")
	}
}
