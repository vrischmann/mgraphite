package mgr

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
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

	i.Add(120)

	buf.Reset()

	err = report(nil)
	require.Nil(t, err)
	require.Equal(t, "foobar 170 100\n", buf.String())
}

func ExampleInt() {
	requestsCounter := NewInt("requests")
	requestsCounter.Add(10)

	// It is safe to call Add and Set from multiple concurrent goroutines.
	var wg sync.WaitGroup
	wg.Add(1000)
	for i := int64(0); i < 1000; i++ {
		go func(i int64) {
			requestsCounter.Add(i)
			wg.Done()
		}(i)
	}
	wg.Wait()

	fmt.Printf("%s\n", requestsCounter.Items()[0].Value)
	// Output:
	// 499510
}

func TestFloat(t *testing.T) {
	buf, fn := reset()
	defer fn()

	f := NewFloat("foobar")
	f.Set(50.1)

	timeFn = func() int64 { return 100 }

	err := report(nil)
	require.Nil(t, err)
	require.Equal(t, "foobar 50.1 100\n", buf.String())
}

func ExampleFloat() {
	ratio := NewFloat("ratio")
	ratio.Set(25.8)

	// It is safe to call Add and Set from multiple concurrent goroutines.
	var wg sync.WaitGroup
	wg.Add(1000)
	for i := int64(0); i < 1000; i++ {
		go func() {
			ratio.Add(1)
			wg.Done()
		}()
	}
	wg.Wait()

	fmt.Printf("%s\n", ratio.Items()[0].Value)
	// Output:
	// 1025.8
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
	require.Equal(t, "foobar.f 20.3 540\nfoobar.i 100 540\n", buf.String())
}

func ExampleMap() {
	httpStats := NewMap("mymap")
	var (
		hits200 Int
		hits500 Int
		hits404 Int
	)
	httpStats.Set("hits200", &hits200)
	httpStats.Set("hits500", &hits500)
	httpStats.Set("hits404", &hits404)

	hits200.Set(5404)
	hits500.Set(3)
	hits404.Set(30)

	httpStats.Do(func(key string, v Var) {
		fmt.Printf("%s => %s\n", key, v.Items()[0].Value)
	})
	// Output:
	// hits200 => 5404
	// hits404 => 30
	// hits500 => 3
}

func ExampleMap_Do() {
	m := NewMap("mymap")
	var (
		a Int
		b Int
		c Int
	)
	m.Set("a", &a)
	m.Set("b", &b)
	m.Set("c", &c)

	a.Set(1)
	b.Set(2)
	c.Set(3)

	m.Do(func(key string, v Var) {
		fmt.Printf("%s => %s\n", key, v.Items()[0].Value)
	})
	// Output:
	// a => 1
	// b => 2
	// c => 3
}

func ExampleMap_Init() {
	m := NewMap("handlers")
	var (
		user        Map
		userLogins  Int
		userLogouts Int

		cart         Map
		cartCheckout Int
		cartClear    Int
		cartAdd      Int
	)

	user.Init().Set("logins", &userLogins)
	user.Set("logouts", &userLogouts)

	cart.Init().Set("checkout", &cartCheckout)
	cart.Set("clear", &cartClear)
	cart.Set("add", &cartAdd)

	m.Set("user", &user)
	m.Set("cart", &cart)

	items := m.Items()
	for _, item := range items {
		fmt.Printf("%s => %s\n", item.Key, item.Value)
	}
	// Output:
	// handlers.cart.add => 0
	// handlers.cart.checkout => 0
	// handlers.cart.clear => 0
	// handlers.user.logins => 0
	// handlers.user.logouts => 0
}

func TestMapInMap(t *testing.T) {
	buf, fn := reset()
	defer fn()

	var i1 Int
	i1.Set(10)
	var i2 Int
	i2.Set(500)
	var i3 Int
	i3.Set(209)
	var m1 Map
	m1.Init().Set("i", &i1)
	var m3 Map
	m3.Init().Set("d", &i3)
	var m2 Map
	m2.Init().Set("i", &i2)
	m2.Set("m", &m3)

	m := NewMap("foo")
	m.Set("bar", &m1)
	m.Set("baz", &m2)

	timeFn = func() int64 { return 600 }

	err := report(nil)
	require.Nil(t, err)
	require.Equal(t, "foo.bar.i 10 600\nfoo.baz.i 500 600\nfoo.baz.m.d 209 600\n", buf.String())
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

type customVar struct {
	sync.Mutex

	handlers struct {
		hits struct {
			c200 int
			c404 int
			c500 int
		}
		execTime struct {
			max  int
			min  int
			last int
		}
	}
}

func (c *customVar) Items() []KeyValue {
	c.Lock()
	defer c.Unlock()

	f := strconv.Itoa

	return []KeyValue{
		{"handlers.hits.c200", f(c.handlers.hits.c200)},
		{"handlers.hits.c404", f(c.handlers.hits.c404)},
		{"handlers.hits.c500", f(c.handlers.hits.c500)},
		{"handlers.execTime.max", f(c.handlers.execTime.max)},
		{"handlers.execTime.min", f(c.handlers.execTime.min)},
		{"handlers.execTime.last", f(c.handlers.execTime.last)},
	}
}

func TestCustomVar(t *testing.T) {
	buf, fn := reset()
	defer fn()

	var cv customVar
	cv.handlers.hits.c200 = 10
	cv.handlers.hits.c404 = 50
	cv.handlers.hits.c500 = 303
	cv.handlers.execTime.max = 30000
	cv.handlers.execTime.min = 300
	cv.handlers.execTime.last = 10000

	Publish(&cv)

	timeFn = func() int64 { return 606 }

	err := report(nil)
	require.Nil(t, err)

	scanner := bufio.NewScanner(buf)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	require.Nil(t, scanner.Err())

	require.Equal(t, "handlers.hits.c200 10 606", lines[0])
	require.Equal(t, "handlers.hits.c404 50 606", lines[1])
	require.Equal(t, "handlers.hits.c500 303 606", lines[2])
	require.Equal(t, "handlers.execTime.max 30000 606", lines[3])
	require.Equal(t, "handlers.execTime.min 300 606", lines[4])
	require.Equal(t, "handlers.execTime.last 10000 606", lines[5])
}
