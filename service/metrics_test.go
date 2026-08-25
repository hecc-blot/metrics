package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metricsConf "github.com/hecc-blot/metrics/config"
)

// TestNewMetrics_DefaultConfig 缺省配置下返回可用实例与采集端点。
func TestNewMetrics_DefaultConfig(t *testing.T) {
	m := NewMetrics(nil)
	require.NotNil(t, m)
	assert.NotNil(t, m.Handler())
}

// TestMetrics_HandlerServesPrometheusText 采集端点输出 Prometheus 文本格式，
// 且包含业务指标、Go 运行时指标与进程指标。
func TestMetrics_HandlerServesPrometheusText(t *testing.T) {
	m := NewMetrics(nil)
	c := m.NewCounter("requests_total", "请求总数", "method")
	c.WithLabelValues("GET").Inc()
	c.WithLabelValues("GET").Inc()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, req)

	body := rec.Body.String()
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, body, "requests_total")
	assert.Contains(t, body, `method="GET"`)
	assert.Contains(t, body, "go_goroutines") // Go 运行时指标
	assert.Contains(t, body, "process_")      // 进程指标
}

// TestMetrics_NamespacePrefix 命名空间前缀应用到业务指标与进程指标。
func TestMetrics_NamespacePrefix(t *testing.T) {
	m := NewMetrics(&metricsConf.Config{Namespace: "demo"})
	m.NewCounter("requests_total", "请求总数").WithLabelValues().Inc()

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	body := rec.Body.String()
	assert.Contains(t, body, "demo_requests_total")
	assert.Contains(t, body, "demo_process_")
}

// TestMetrics_DuplicateNameReuses 同名指标复用已注册实例，不 panic。
func TestMetrics_DuplicateNameReuses(t *testing.T) {
	m := NewMetrics(nil)
	require.NotPanics(t, func() {
		_ = m.NewCounter("dup_total", "a")
		_ = m.NewCounter("dup_total", "b") // 第二次复用，不重复注册
		_ = m.NewGauge("dup_gauge", "a")
		_ = m.NewGauge("dup_gauge", "b")
		_ = m.NewHistogram("dup_hist", "a")
		_ = m.NewHistogram("dup_hist", "b")
	})
}

// TestMetrics_GaugeAndHistogram 仪表可设值、直方图可观测，并正确输出。
func TestMetrics_GaugeAndHistogram(t *testing.T) {
	m := NewMetrics(nil)
	g := m.NewGauge("connections", "当前连接数")
	g.WithLabelValues().Set(42)
	h := m.NewHistogram("latency_seconds", "延迟")
	h.WithLabelValues().Observe(0.25)

	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	body := rec.Body.String()
	assert.Contains(t, body, "connections 42")
	assert.Contains(t, body, "latency_seconds_count")
}

// TestMetrics_NormalizeDefaultPath 默认端点路径为 /metrics。
func TestMetrics_NormalizeDefaultPath(t *testing.T) {
	assert.Equal(t, "/metrics", metricsConf.Normalize(metricsConf.Config{}).Path)
	assert.Equal(t, "/custom", metricsConf.Normalize(metricsConf.Config{Path: "/custom"}).Path)
	assert.Equal(t, "", metricsConf.Normalize(metricsConf.Config{}).Namespace)
}
