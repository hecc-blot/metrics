package mocks

import (
	"net/http"

	metricsContract "github.com/hecc-blot/metrics/contract"
)

// MockMetrics 是 IMetrics 接口的 mock 实现。
// 通过 HandlerFn / NewCounterFn / NewGaugeFn / NewHistogramFn 定制行为，
// 未设置时返回内存记录型 mock（见 MockCounterVec 等），便于断言标签与调用次数。
type MockMetrics struct {
	HandlerFn      func() http.Handler
	NewCounterFn   func(name, help string, labelNames ...string) metricsContract.ICounterVec
	NewGaugeFn     func(name, help string, labelNames ...string) metricsContract.IGaugeVec
	NewHistogramFn func(name, help string, labelNames ...string) metricsContract.IHistogramVec

	// CounterVecs 等按 name 记录默认 mock，便于在测试里回查并断言。
	CounterVecs   map[string]*MockCounterVec
	GaugeVecs     map[string]*MockGaugeVec
	HistogramVecs map[string]*MockHistogramVec
}

func (m *MockMetrics) Handler() http.Handler {
	if m.HandlerFn != nil {
		return m.HandlerFn()
	}
	return http.NotFoundHandler()
}

func (m *MockMetrics) NewCounter(name, help string, labelNames ...string) metricsContract.ICounterVec {
	if m.NewCounterFn != nil {
		return m.NewCounterFn(name, help, labelNames...)
	}
	v := &MockCounterVec{LabelNames: labelNames}
	if m.CounterVecs == nil {
		m.CounterVecs = make(map[string]*MockCounterVec)
	}
	m.CounterVecs[name] = v
	return v
}

func (m *MockMetrics) NewGauge(name, help string, labelNames ...string) metricsContract.IGaugeVec {
	if m.NewGaugeFn != nil {
		return m.NewGaugeFn(name, help, labelNames...)
	}
	v := &MockGaugeVec{LabelNames: labelNames}
	if m.GaugeVecs == nil {
		m.GaugeVecs = make(map[string]*MockGaugeVec)
	}
	m.GaugeVecs[name] = v
	return v
}

func (m *MockMetrics) NewHistogram(name, help string, labelNames ...string) metricsContract.IHistogramVec {
	if m.NewHistogramFn != nil {
		return m.NewHistogramFn(name, help, labelNames...)
	}
	v := &MockHistogramVec{LabelNames: labelNames}
	if m.HistogramVecs == nil {
		m.HistogramVecs = make(map[string]*MockHistogramVec)
	}
	m.HistogramVecs[name] = v
	return v
}

// MockCounter 记录 Inc / Add 调用。
type MockCounter struct {
	IncCalls int
	Added    []float64
}

func (m *MockCounter) Inc()             { m.IncCalls++ }
func (m *MockCounter) Add(delta float64) { m.Added = append(m.Added, delta) }

// MockCounterVec 记录 WithLabelValues 调用，返回共享的 MockCounter。
type MockCounterVec struct {
	LabelNames []string
	Calls      [][]string
	Counter    *MockCounter
}

func (m *MockCounterVec) WithLabelValues(lvs ...string) metricsContract.ICounter {
	m.Calls = append(m.Calls, lvs)
	if m.Counter == nil {
		m.Counter = &MockCounter{}
	}
	return m.Counter
}

// MockGauge 记录 Inc / Dec / Add / Set 调用。
type MockGauge struct {
	IncCalls int
	DecCalls int
	Added    []float64
	SetValue *float64
}

func (m *MockGauge) Inc()             { m.IncCalls++ }
func (m *MockGauge) Dec()             { m.DecCalls++ }
func (m *MockGauge) Add(delta float64) { m.Added = append(m.Added, delta) }
func (m *MockGauge) Set(value float64) {
	v := value
	m.SetValue = &v
}

// MockGaugeVec 记录 WithLabelValues 调用，返回共享的 MockGauge。
type MockGaugeVec struct {
	LabelNames []string
	Calls      [][]string
	Gauge      *MockGauge
}

func (m *MockGaugeVec) WithLabelValues(lvs ...string) metricsContract.IGauge {
	m.Calls = append(m.Calls, lvs)
	if m.Gauge == nil {
		m.Gauge = &MockGauge{}
	}
	return m.Gauge
}

// MockHistogram 记录 Observe 调用。
type MockHistogram struct {
	Observed []float64
}

func (m *MockHistogram) Observe(value float64) { m.Observed = append(m.Observed, value) }

// MockHistogramVec 记录 WithLabelValues 调用，返回共享的 MockHistogram。
type MockHistogramVec struct {
	LabelNames []string
	Calls      [][]string
	Histogram  *MockHistogram
}

func (m *MockHistogramVec) WithLabelValues(lvs ...string) metricsContract.IHistogram {
	m.Calls = append(m.Calls, lvs)
	if m.Histogram == nil {
		m.Histogram = &MockHistogram{}
	}
	return m.Histogram
}

// 编译期断言：确保 mock 完整实现接口。
var _ metricsContract.IMetrics = (*MockMetrics)(nil)
var _ metricsContract.ICounter = (*MockCounter)(nil)
var _ metricsContract.ICounterVec = (*MockCounterVec)(nil)
var _ metricsContract.IGauge = (*MockGauge)(nil)
var _ metricsContract.IGaugeVec = (*MockGaugeVec)(nil)
var _ metricsContract.IHistogram = (*MockHistogram)(nil)
var _ metricsContract.IHistogramVec = (*MockHistogramVec)(nil)
