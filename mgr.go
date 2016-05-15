package mgr

import (
	"bytes"
	"errors"
	"io"
	"log"
	"math"
	"net"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Config allows you to alter the behaviour of mgr.
type Config struct {
	// Interval at which mgr exports data to Graphite.
	Interval time.Duration
	// Addr address of the Graphite server (with the port).
	Addr string
	// Logger allows you to override the logger used to report errors.
	Logger func(format string, args ...interface{})
}

var (
	// ErrInvalidConfig is returned when the configuration is invalid (missing Graphite address mainly).
	ErrInvalidConfig = errors.New("invalid config")

	// DiscardLogger can be used as a Logger if you want to silence the errors.
	DiscardLogger = func(format string, args ...interface{}) {}

	vars struct {
		sync.Mutex
		l []Var
	}
)

type Var interface {
	Items() []KeyValue
}

// Func implements Var by calling the function.
type Func func() []KeyValue

func (f Func) Items() []KeyValue { return f() }

func init() {
	Publish(Func(readMemStats))
}

// KeyValue represents a single Graphite metric.
type KeyValue struct {
	Key   string
	Value string
}

// Int is a 64-bit integer variable that satisfies the Var interface.
type Int struct {
	key string
	i   int64
}

func (i *Int) Items() []KeyValue {
	return []KeyValue{{
		Key:   i.key,
		Value: strconv.FormatInt(atomic.LoadInt64(&i.i), 10),
	}}
}
func (i *Int) Add(delta int64) { atomic.AddInt64(&i.i, delta) }
func (i *Int) Set(val int64)   { atomic.StoreInt64(&i.i, val) }

// NewInt creates a Int and publishes it.
func NewInt(name string) *Int {
	i := &Int{key: name}
	Publish(i)

	return i
}

// Float is a 64-bit float variable that satisfies the Var interface.
type Float struct {
	key string
	f   uint64
}

func (f *Float) Items() []KeyValue {
	return []KeyValue{{
		Key:   f.key,
		Value: strconv.FormatFloat(math.Float64frombits(atomic.LoadUint64(&f.f)), 'g', -1, 64),
	}}
}

func (f *Float) Add(delta float64) {
	for {
		cur := atomic.LoadUint64(&f.f)
		curVal := math.Float64frombits(cur)
		nxtVal := curVal + delta
		nxt := math.Float64bits(nxtVal)
		if atomic.CompareAndSwapUint64(&f.f, cur, nxt) {
			return
		}
	}
}

func (f *Float) Set(val float64) { atomic.StoreUint64(&f.f, math.Float64bits(val)) }

// NewFloat creates a Float and publishes it.
func NewFloat(name string) *Float {
	f := &Float{key: name}
	Publish(f)

	return f
}

// TODO(vincent): does this make sense when exporting to graphite ? I don't think so.
// type String struct {
// 	sync.RWMutex
// 	key string
// 	s   string
// }
//
// func NewString(name string) *String {
// 	s := &String{key: name}
// 	Publish(s)
//
// 	return s
// }
//
// func (v *String) Items() []KeyValue {
// 	v.RLock()
// 	defer v.RUnlock()
//
// 	return []KeyValue{{
// 		Key:   v.key,
// 		Value: v.s,
// 	}}
// }
//
// func (v *String) Set(value string) {
// 	v.Lock()
// 	defer v.Unlock()
// 	v.s = value
// }

// Map is a string-to-Var map variable that satisfies the Var interface.
type Map struct {
	mu   sync.Mutex
	key  string
	m    map[string]Var
	keys []string
}

// NewMap creates a new Map and publishes it.
func NewMap(name string) *Map {
	m := &Map{
		key: name,
		m:   make(map[string]Var),
	}
	Publish(m)

	return m
}

func flattenMap(prefix string, m map[string]Var) (res []KeyValue) {
	for k, val := range m {
		key := prefix + "." + k

		switch v := val.(type) {
		case *Map:
			res = append(res, flattenMap(key, v.m)...)
		default:
			for _, item := range v.Items() {
				res = append(res, KeyValue{
					Key:   key,
					Value: item.Value,
				})
			}
		}
	}
	return
}

func (m *Map) Items() []KeyValue {
	m.mu.Lock()
	defer m.mu.Unlock()

	return flattenMap(m.key, m.m)
}

func (m *Map) Set(key string, val Var) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.m[key] = val

	// Update the sorted keys
	m.keys = make([]string, len(m.m))
	i := 0
	for k, _ := range m.m {
		m.keys[i] = k
		i++
	}

	sort.Strings(m.keys)
}

// Do calls f for each entry in the map. The map is locked during the iteration, but existing entries may be concurrently updated.
func (m *Map) Do(fn func(key string, v Var)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, k := range m.keys {
		v := m.m[k]
		fn(k, v)
	}
}

// Publish declares a named exported variable.
func Publish(v Var) {
	vars.Lock()
	vars.l = append(vars.l, v)
	vars.Unlock()
}

// Do calls f for each exported variable.
// The global variable list is locked during the iteration, but existing entries may be concurrently updated.
func Do(fn func(v Var)) {
	vars.Lock()
	for _, v := range vars.l {
		fn(v)
	}
	vars.Unlock()
}

func Export(config *Config) error {
	if config == nil {
		return ErrInvalidConfig
	}

	if config.Interval == 0 {
		config.Interval = 1 * time.Minute
	}
	if config.Logger == nil {
		config.Logger = log.Printf
	}

	ticker := time.NewTicker(config.Interval)
	for range ticker.C {
		if err := report(config); err != nil {
			config.Logger("unable to report data. err=%v", err)
		}
	}

	return nil
}

type dialFunc func(config *Config) (io.Writer, error)
type timeFunc func() int64

var (
	dialFn dialFunc = defaultDial
	timeFn timeFunc = defaulTimeNow
	conn   io.Writer

	bufPool = sync.Pool{
		New: func() interface{} { return new(bytes.Buffer) },
	}
)

func defaultDial(config *Config) (io.Writer, error) {
	return net.Dial("tcp", config.Addr)
}

func defaulTimeNow() int64 {
	return time.Now().UnixNano() / int64(time.Second)
}

func appendMetric(buf *bytes.Buffer, v Var) {
	for _, kv := range v.Items() {
		buf.WriteString(kv.Key + " " + kv.Value + " ")
		buf.WriteString(strconv.FormatInt(timeFn(), 10))
		buf.WriteRune('\n')
	}
}

func report(config *Config) error {
	if conn == nil {
		var err error
		conn, err = dialFn(config)
		if err != nil {
			return err
		}
	}

	buf := bufPool.Get().(*bytes.Buffer)
	defer bufPool.Put(buf)

	Do(func(v Var) { appendMetric(buf, v) })

	_, err := io.Copy(conn, buf)
	if err != nil {
		conn = nil
		return err
	}

	return nil
}

var (
	_ Var = (Func)(nil)
	_ Var = (*Int)(nil)
	_ Var = (*Float)(nil)
	_ Var = (*Map)(nil)
)
