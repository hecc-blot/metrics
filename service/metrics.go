package service

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	metricsConf "github.com/hecc-blot/metrics/config"
	metricsContract "github.com/hecc-blot/metrics/contract"
)

// metricsSvc 基于 Prometheus 的监控指标实现。
type metricsSvc struct {
	registry  *prometheus.Registry
	handler   http.Handler
	namespace string

	// 缓存已创建的 Vec，避免同名指标重复注册导致 panic。
	mu         sync.Mutex
	counters   map[string]*prometheus.CounterVec
	gauges     map[string]*prometheus.GaugeVec
	histograms map[string]*prometheus.HistogramVec
}

// NewMetrics 创建监控指标服务。cfg 可为 nil，缺省使用默认端点 /metrics。
func NewMetrics(cfg *metricsConf.Config) metricsContract.IMetrics {
	var c metricsConf.Config
	if cfg != nil {
		c = *cfg
	}
	c = metricsConf.Normalize(c)

	registry := prometheus.NewRegistry()
	// Go 运行时与进程级指标（CPU/内存/GC/句柄等），生产就绪必暴露，供容量与异常告警。
	registry.MustRegister(collectors.NewGoCollector())
	registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{Namespace: c.Namespace}))

	return &metricsSvc{
		registry:   registry,
		handler:    promhttp.HandlerFor(registry, promhttp.HandlerOpts{}),
		namespace:  c.Namespace,
		counters:   make(map[string]*prometheus.CounterVec),
		gauges:     make(map[string]*prometheus.GaugeVec),
		histograms: make(map[string]*prometheus.HistogramVec),
	}
}

// Handler 返回 Prometheus 采集端点。
func (s *metricsSvc) Handler() http.Handler { return s.handler }

// NewCounter 创建带标签计数器，同名复用。
func (s *metricsSvc) NewCounter(name, help string, labelNames ...string) metricsContract.ICounterVec {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.counters[name]; ok {
		return &counterVec{v}
	}
	v := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: s.fullName(name),
		Help: help,
	}, labelNames)
	s.registry.MustRegister(v)
	s.counters[name] = v
	return &counterVec{v}
}

// NewGauge 创建带标签仪表，同名复用。
func (s *metricsSvc) NewGauge(name, help string, labelNames ...string) metricsContract.IGaugeVec {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.gauges[name]; ok {
		return &gaugeVec{v}
	}
	v := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: s.fullName(name),
		Help: help,
	}, labelNames)
	s.registry.MustRegister(v)
	s.gauges[name] = v
	return &gaugeVec{v}
}

// NewHistogram 创建带标签直方图，同名复用。
func (s *metricsSvc) NewHistogram(name, help string, labelNames ...string) metricsContract.IHistogramVec {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.histograms[name]; ok {
		return &histogramVec{v}
	}
	v := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: s.fullName(name),
		Help: help,
	}, labelNames)
	s.registry.MustRegister(v)
	s.histograms[name] = v
	return &histogramVec{v}
}

// fullName 应用命名空间前缀。
func (s *metricsSvc) fullName(name string) string {
	if s.namespace == "" {
		return name
	}
	return s.namespace + "_" + name
}

// —— Vec 适配器：把 prometheus 的具体类型收敛到 contract 接口 ——

type counterVec struct{ v *prometheus.CounterVec }

func (c *counterVec) WithLabelValues(lvs ...string) metricsContract.ICounter {
	return c.v.WithLabelValues(lvs...)
}

type gaugeVec struct{ v *prometheus.GaugeVec }

func (g *gaugeVec) WithLabelValues(lvs ...string) metricsContract.IGauge {
	return g.v.WithLabelValues(lvs...)
}

type histogramVec struct{ v *prometheus.HistogramVec }

func (h *histogramVec) WithLabelValues(lvs ...string) metricsContract.IHistogram {
	return h.v.WithLabelValues(lvs...)
}
