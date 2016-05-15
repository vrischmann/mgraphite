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

type Config struct {
	Interval time.Duration
	Addr     string
	Logger   func(format string, args ...interface{})
}

var (
	ErrInvalidConfig = errors.New("invalid config")

	DiscardLogger = func(format string, args ...interface{}) {}

	vars struct {
		sync.Mutex
		l []Var
	}
)

type Func func() []KeyValue

func (f Func) Items() []KeyValue { return f() }

func init() {
	Publish(Func(readMemStats))
}

type Var interface {
	Items() []KeyValue
}

type KeyValue struct {
	Key   string
	Value string
}

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

func NewInt(name string) *Int {
	i := &Int{key: name}
	Publish(i)

	return i
}

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

type Map struct {
	mu   sync.Mutex
	key  string
	m    map[string]Var
	keys []string
}

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

func (m *Map) Do(fn func(key string, v Var)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, k := range m.keys {
		v := m.m[k]
		fn(k, v)
	}
}

func Publish(v Var) {
	vars.Lock()
	vars.l = append(vars.l, v)
	vars.Unlock()
}

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
			log.Printf("unable to report data. err=%v", err)
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
