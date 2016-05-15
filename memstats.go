package mgr

import (
	"runtime"
	"strconv"
)

// MemStats is a function that returns data from runtime.MemStats.
// It is not published by default; you need to publish it yourself.
// The reason behind this is because it's not a free operation to read runtime memory statistics.
//
// Here is how to publish it:
//
//     mgr.publish(mgr.Func(MemStats()))
func MemStats() []KeyValue {
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)

	mostRecentPauseNs := stats.PauseNs[(stats.NumGC+255)%256]
	mostRecentPauseEnd := stats.PauseEnd[(stats.NumGC+255)%256]

	return []KeyValue{
		{"memstats.Alloc", strconv.FormatUint(stats.Alloc, 10)},
		{"memstats.TotalAlloc", strconv.FormatUint(stats.TotalAlloc, 10)},
		{"memstats.Sys", strconv.FormatUint(stats.Sys, 10)},
		{"memstats.Lookups", strconv.FormatUint(stats.Lookups, 10)},
		{"memstats.Mallocs", strconv.FormatUint(stats.Mallocs, 10)},
		{"memstats.Frees", strconv.FormatUint(stats.Frees, 10)},
		{"memstats.HeapAlloc", strconv.FormatUint(stats.HeapAlloc, 10)},
		{"memstats.HeapSys", strconv.FormatUint(stats.HeapSys, 10)},
		{"memstats.HeapIdle", strconv.FormatUint(stats.HeapIdle, 10)},
		{"memstats.HeapInuse", strconv.FormatUint(stats.HeapInuse, 10)},
		{"memstats.HeapReleased", strconv.FormatUint(stats.HeapReleased, 10)},
		{"memstats.HeapObjects", strconv.FormatUint(stats.HeapObjects, 10)},
		{"memstats.StackInuse", strconv.FormatUint(stats.StackInuse, 10)},
		{"memstats.StackSys", strconv.FormatUint(stats.StackSys, 10)},
		{"memstats.MSpanInuse", strconv.FormatUint(stats.MSpanInuse, 10)},
		{"memstats.MSpanSys", strconv.FormatUint(stats.MSpanSys, 10)},
		{"memstats.MCacheInuse", strconv.FormatUint(stats.MCacheInuse, 10)},
		{"memstats.MCacheSys", strconv.FormatUint(stats.MCacheSys, 10)},
		{"memstats.BuckHashSys", strconv.FormatUint(stats.BuckHashSys, 10)},
		{"memstats.GCSys", strconv.FormatUint(stats.GCSys, 10)},
		{"memstats.OtherSys", strconv.FormatUint(stats.OtherSys, 10)},
		{"memstats.NextGC", strconv.FormatUint(stats.NextGC, 10)},
		{"memstats.LastGC", strconv.FormatUint(stats.LastGC, 10)},
		{"memstats.PauseTotalNs", strconv.FormatUint(stats.PauseTotalNs, 10)},
		{"memstats.MostRecentPauseNs", strconv.FormatUint(mostRecentPauseNs, 10)},
		{"memstats.MostRecentPauseEnd", strconv.FormatUint(mostRecentPauseEnd, 10)},
		{"memstats.NumGC", strconv.FormatUint(uint64(stats.NumGC), 10)},
		{"memstats.GCCPUFraction", strconv.FormatFloat(stats.GCCPUFraction, 'g', -1, 64)},
		{"memstats.EnableGC", strconv.FormatBool(stats.EnableGC)},
		{"memstats.DebugGC", strconv.FormatBool(stats.DebugGC)},
	}
}
