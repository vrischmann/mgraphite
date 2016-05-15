package mgr

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func reset() (*bytes.Buffer, func()) {
	conn = nil
	vars.l = nil

	buf := new(bytes.Buffer)
	dialFn = func(_ *Config) (io.Writer, error) {
		return buf, nil
	}
	resetDialFn := func() { dialFn = defaultDial }

	return buf, resetDialFn
}

func TestEmpty(t *testing.T) {
	buf, fn := reset()
	defer fn()

	err := report(nil)
	require.Nil(t, err)
	require.Equal(t, 0, buf.Len())
}

func TestInt(t *testing.T) {
	buf, fn := reset()
	defer fn()

	i := NewInt("foobar")
	i.Set(50)

	timeFn = func() int64 { return 100 }

	err := report(nil)
	require.Nil(t, err)
	require.Equal(t, "foobar 50 100\n", buf.String())
}

func TestConcurrentInt(t *testing.T) {
	buf, fn := reset()
	defer fn()

	i := NewInt("foobar")
	var wg sync.WaitGroup
	wg.Add(4000)
	for j := 0; j < 4000; j++ {
		go func() {
			i.Add(1)
			wg.Done()
		}()
	}
	wg.Wait()

	timeFn = func() int64 { return 100 }

	err := report(nil)
	require.Nil(t, err)
	require.Equal(t, "foobar 4000 100\n", buf.String())
}

func TestMultipleConcurrent(t *testing.T) {
	buf, fn := reset()
	defer fn()

	i := NewInt("foobar.int")
	f := NewFloat("foobar.float")

	var wg sync.WaitGroup
	wg.Add(4000)
	for j := 0; j < 4000; j++ {
		go func() {
			i.Add(1)
			f.Add(1.0)
			wg.Done()
		}()
	}
	wg.Wait()

	timeFn = func() int64 { return 100 }

	err := report(nil)
	require.Nil(t, err)
	require.Equal(t, "foobar.int 4000 100\nfoobar.float 4000 100\n", buf.String())
}

func TestMap(t *testing.T) {
	buf, fn := reset()
	defer fn()

	var (
		i Int
		f Float
	)

	i.Set(100)
	f.Set(20.3)

	m := NewMap("foobar")
	m.Set("i", &i)
	m.Set("f", &f)

	timeFn = func() int64 { return 540 }

	err := report(nil)
	require.Nil(t, err)
	require.Equal(t, "foobar.i 100 540\nfoobar.f 20.3 540\n", buf.String())
}

func TestMemstats(t *testing.T) {
	buf, fn := reset()
	defer fn()

	vars.l = append(vars.l, Func(MemStats))

	i := NewInt("foobar.i")
	i.Set(3050)

	timeFn = func() int64 { return 606 }

	err := report(nil)
	require.Nil(t, err)

	scanner := bufio.NewScanner(buf)
	sawFoobar := false
	for scanner.Scan() {
		line := scanner.Text()
		require.True(t, strings.HasSuffix(line, "606"))
		if strings.HasPrefix(line, "foobar.i 3050") {
			sawFoobar = true
		}
	}
	require.True(t, sawFoobar)
}
