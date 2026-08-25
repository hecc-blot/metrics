package service

import (
	"strconv"
	"time"

	iCoreApi "github.com/hecc-blot/framework/contract/api"
	metricsContract "github.com/hecc-blot/metrics/contract"

	"github.com/gin-gonic/gin"
)

// HttpMetricsMiddleware 为每个 HTTP 请求记录请求总数与耗时分布，
// 支撑 QPS / 延迟 / 错误率三类上线告警。
type HttpMetricsMiddleware struct {
	metrics metricsContract.IMetrics
}

// NewHttpMiddleware 创建 HTTP 监控指标中间件，供 api 在组装阶段注册。
func NewHttpMiddleware(m metricsContract.IMetrics) iCoreApi.IMiddleware {
	return &HttpMetricsMiddleware{metrics: m}
}

func (h *HttpMetricsMiddleware) Middleware() any {
	// 指标在注册阶段创建一次、跨请求复用（标签按 method/path/status 细分）。
	requests := h.metrics.NewCounter(
		"http_requests_total",
		"HTTP 请求总数",
		"method", "path", "status",
	)
	duration := h.metrics.NewHistogram(
		"http_request_duration_seconds",
		"HTTP 请求耗时分布（秒）",
		"method", "path", "status",
	)

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		method := c.Request.Method
		path := routePattern(c)
		status := strconv.Itoa(c.Writer.Status())

		duration.WithLabelValues(method, path, status).Observe(time.Since(start).Seconds())
		requests.WithLabelValues(method, path, status).Inc()
	}
}

// routePattern 取路由模板（如 /account/:id）作为 path 标签，
// 避免把真实 ID 打散成高基数序列。未匹配路由（FullPath 为空，如 404）回退为 "unmatched"。
func routePattern(c *gin.Context) string {
	if p := c.FullPath(); p != "" {
		return p
	}
	return "unmatched"
}
