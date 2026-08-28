package metrics

import "net/http"

// ICounter 计数器：单调递增，用于请求总数、错误总数等。
type ICounter interface {
	Inc()              // 自增 1
	Add(delta float64) // 增加 delta
}

// IGauge 仪表：可增可减可设，用于连接数、队列长度等瞬时值。
type IGauge interface {
	Inc()
	Dec()
	Add(delta float64)
	Set(value float64)
}

// IHistogram 直方图：记录观测值分布，用于延迟、耗时等。
type IHistogram interface {
	Observe(value float64)
}

// ICounterVec 带标签的计数器：按标签值定位到具体计数器。
type ICounterVec interface {
	WithLabelValues(lvs ...string) ICounter
}

// IGaugeVec 带标签的仪表。
type IGaugeVec interface {
	WithLabelValues(lvs ...string) IGauge
}

// IHistogramVec 带标签的直方图。
type IHistogramVec interface {
	WithLabelValues(lvs ...string) IHistogram
}

// IMetrics 监控指标：注册指标并暴露 Prometheus 采集端点。
type IMetrics interface {
	// Handler 返回 Prometheus 采集端点，供挂载到 HTTP 路由
	// （如 api.Handle(http.MethodGet, cfg.Path, m.Handler())）。
	Handler() http.Handler

	// NewCounter 创建带标签计数器。labelNames 为标签维度，与 WithLabelValues 一一对应。
	// 同名指标复用已注册实例，重复调用不 panic。
	NewCounter(name, help string, labelNames ...string) ICounterVec
	NewGauge(name, help string, labelNames ...string) IGaugeVec
	NewHistogram(name, help string, labelNames ...string) IHistogramVec
}
