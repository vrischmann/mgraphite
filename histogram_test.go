package mgr

import (
	"log"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHistogramMean(t *testing.T) {
	h := NewHistogram("foobar", 10000)

	for i := int64(0); i < 10000; i++ {
		h.Record(2)
	}

	require.Equal(t, int64(10000), h.counter)
	h.takeSnapshot()
	require.Equal(t, 2.0, h.mean())
}

func TestHistogramMax(t *testing.T) {
	h := NewHistogram("foobar", 10000)

	for i := int64(0); i < 10000; i++ {
		h.Record(i)
	}

	require.Equal(t, int64(10000), h.counter)
	h.takeSnapshot()
	require.Equal(t, int64(9999), h.max())
}

func TestHistogramMin(t *testing.T) {
	h := NewHistogram("foobar", 8)

	for i := int64(2); i < 10; i++ {
		h.Record(i)
	}

	require.Equal(t, int64(8), h.counter)
	h.takeSnapshot()
	require.Equal(t, int64(2), h.min())
}

func TestHistogramStddev(t *testing.T) {
	h := NewHistogram("foobar", 8)

	h.Record(2)
	h.Record(4)
	h.Record(4)
	h.Record(4)
	h.Record(5)
	h.Record(5)
	h.Record(7)
	h.Record(9)

	h.takeSnapshot()

	require.Equal(t, float64(2), h.stddev())
}

func TestHistogramPercentile(t *testing.T) {
	h := NewHistogram("foobar", 1000)
	for i := 0; i < 300; i++ {
		h.Record(100)
	}
	for i := 0; i < 300; i++ {
		h.Record(500)
	}
	for i := 0; i < 300; i++ {
		h.Record(800)
	}
	for i := 0; i < 100; i++ {
		h.Record(3000)
	}

	h.takeSnapshot()

	require.Equal(t, int64(3000), h.percentile(95))
	require.Equal(t, int64(3000), h.percentile(99))
	require.Equal(t, int64(3000), h.percentile(100))
}

func TestHistogram(t *testing.T) {
	h := NewHistogram("foobar", 8)

	for i := int64(0); i < 10; i++ {
		h.Record(rand.Int63n(100))
	}

	items := h.Items()
	log.Printf("items: %v", items)
	// TODO(vincent): test this somehow
}
