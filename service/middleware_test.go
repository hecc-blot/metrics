package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestEngine 组装带指标中间件的 gin 引擎，挂载采集端点。
func newTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	m := NewMetrics(nil)

	r := gin.New()
	mw := gin.HandlerFunc(NewHttpMiddleware(m).Middleware().(func(*gin.Context)))
	r.Use(mw)
	r.GET("/metrics", gin.WrapH(m.Handler()))
	return r
}

// TestHttpMiddleware_RecordsRequestMetrics 中间件记录请求总数与耗时，path 用路由模板避免高基数。
func TestHttpMiddleware_RecordsRequestMetrics(t *testing.T) {
	r := newTestEngine(t)
	r.GET("/account/:id", func(c *gin.Context) { c.Status(http.StatusOK) })

	// 发起业务请求
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/account/123", nil))
	assert.Equal(t, http.StatusOK, w.Code)

	// 抓取指标
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()

	assert.Contains(t, body, "http_requests_total")
	assert.Contains(t, body, `method="GET"`)
	assert.Contains(t, body, `path="/account/:id"`) // 路由模板，而非 /account/123
	assert.Contains(t, body, `status="200"`)
	assert.Contains(t, body, "http_request_duration_seconds_count")
}

// TestHttpMiddleware_UnmatchedRoute 未匹配路由回退 path="unmatched" 并记录 404。
func TestHttpMiddleware_UnmatchedRoute(t *testing.T) {
	r := newTestEngine(t)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/nope", nil))
	assert.Equal(t, http.StatusNotFound, w.Code)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()

	assert.Contains(t, body, `path="unmatched"`)
	assert.Contains(t, body, `status="404"`)
}

// TestHttpMiddleware_CountsEachStatus 不同状态码各自计数。
func TestHttpMiddleware_CountsEachStatus(t *testing.T) {
	r := newTestEngine(t)
	r.GET("/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/err", func(c *gin.Context) { c.Status(http.StatusInternalServerError) })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ping", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/err", nil))

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()

	require.Contains(t, body, `path="/ping"`)
	require.Contains(t, body, `path="/err"`)
	assert.Contains(t, body, `status="500"`)
}
