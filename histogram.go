package mgr

import (
	"math"
	"sort"
	"strconv"
	"sync"
	"time"
)

type Histogram struct {
	key    string
	Buffer []int64

	mu       sync.Mutex
	counter  int64
	snapshot []int64
}

func NewHistogram(name string, bufferSize int) *Histogram {
	h := &Histogram{
		key:      name,
		Buffer:   make([]int64, bufferSize),
		snapshot: make([]int64, bufferSize),
	}
	Publish(h)

	return h
}

func (h *Histogram) Init(bufferSize int) *Histogram {
	h.Buffer = make([]int64, bufferSize)
	h.snapshot = make([]int64, bufferSize)

	return h
}

func (h *Histogram) takeSnapshot() {
	h.mu.Lock()

	// TODO(vincent): maybe we'll need to have multiple snapshots to protect
	// from concurrent Items() calls.
	copy(h.snapshot, h.Buffer)

	h.mu.Unlock()
}

func (h *Histogram) mean() float64 {
	sum := int64(0)
	v := h.snapshot

	for _, val := range v {
		sum += val
	}

	return float64(sum) / float64(len(v))
}

func (h *Histogram) max() (res int64) {
	for _, val := range h.snapshot {
		if val > res {
			res = val
		}
	}
	return
}

func (h *Histogram) min() (res int64) {
	if len(h.snapshot) == 0 {
		return 0
	}

	res = math.MaxInt64
	for _, val := range h.snapshot {
		if val < res {
			res = val
		}
	}
	return
}

func (h *Histogram) stddev() float64 {
	if len(h.snapshot) == 0 {
		return 0.0
	}

	m := h.mean()

	var sum float64
	for _, val := range h.snapshot {
		d := float64(val) - m
		sum += d * d
	}

	variance := sum / float64(len(h.snapshot))

	return math.Sqrt(variance)
}

type int64slice []int64

func (s int64slice) Len() int           { return len(s) }
func (s int64slice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s int64slice) Less(i, j int) bool { return s[i] < s[j] }

func (h *Histogram) percentile(p float64) int64 {
	// https://en.wikipedia.org/wiki/Percentile#The_Nearest_Rank_method
	n := int(math.Ceil(p / 100 * float64(len(h.snapshot)-1)))
	sort.Sort(int64slice(h.snapshot))

	return h.snapshot[n]
}

func (h *Histogram) Record(val int64) {
	h.mu.Lock()

	idx := int(h.counter % int64(len(h.Buffer)))
	h.Buffer[idx] = val
	h.counter++

	h.mu.Unlock()
}

func (h *Histogram) RecordSince(t time.Time) {
	h.Record(int64(time.Since(t)))
}

func (h *Histogram) Items() []KeyValue {
	h.takeSnapshot()

	n := func(s string) string { return h.key + "." + s }
	f := func(f float64) string { return strconv.FormatFloat(f, 'g', 5, 64) }
	i := func(i int64) string { return strconv.FormatInt(i, 10) }

	return []KeyValue{
		{n("mean"), f(h.mean())},
		{n("max"), i(h.max())},
		{n("min"), i(h.min())},
		{n("stddev"), f(h.stddev())},
		{n("p50"), i(h.percentile(50))},
		{n("p75"), i(h.percentile(75))},
		{n("p90"), i(h.percentile(90))},
		{n("p95"), i(h.percentile(95))},
		{n("p98"), i(h.percentile(98))},
		{n("p99"), i(h.percentile(99))},
		{n("p999"), i(h.percentile(99.9))},
		{n("p9999"), i(h.percentile(99.99))},
	}
}
