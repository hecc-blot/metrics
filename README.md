# hecc-blot-metrics

面向接口的监控指标组件：注册 Counter / Gauge / Histogram 指标，暴露 Prometheus 采集端点，并内置 HTTP 中间件自动采集 QPS / 延迟 / 错误率，支撑上线告警。

## 安装

```bash
go get github.com/hecc-blot/metrics
```

## 接口定义

```go
import metricsContract "github.com/hecc-blot/metrics/contract"

type IMetrics interface {
    Handler() http.Handler                                            // Prometheus 采集端点
    NewCounter(name, help string, labels ...string) ICounterVec      // 计数器：请求数、错误数
    NewGauge(name, help string, labels ...string) IGaugeVec          // 仪表：连接数、队列长度
    NewHistogram(name, help string, labels ...string) IHistogramVec  // 直方图：延迟分布
}
```

`ICounterVec / IGaugeVec / IHistogramVec` 通过 `WithLabelValues(...)` 定位到具体指标，再调用 `Inc / Add / Dec / Set / Observe`。

## 初始化

```go
import (
    metricsConf "github.com/hecc-blot/metrics/config"
    metrics "github.com/hecc-blot/metrics/service"
)

metricsSvc := metrics.NewMetrics(&metricsConf.Config{Namespace: "myapp"})
```

`cfg` 可为 nil，缺省使用默认端点 `/metrics`。`Namespace` 统一为业务指标与进程指标加前缀。

## 挂载端点与中间件

```go
// 采集端点挂到 HTTP 路由
apiHandle.Handle(http.MethodGet, config.Metrics.Path, metricsSvc.Handler())

// 自动采集 QPS / 延迟 / 错误率（path 用路由模板避免高基数）
apiHandle.Middleware(metrics.NewHttpMiddleware(metricsSvc))
```

中间件自动产出两类指标：

| 指标 | 类型 | 标签 | 用途 |
|------|------|------|------|
| `http_requests_total` | Counter | method / path / status | QPS、错误率（按 status 区分） |
| `http_request_duration_seconds` | Histogram | method / path / status | 延迟分布、P95/P99 |

`path` 标签取路由模板（如 `/account/:id`），未匹配路由回退 `unmatched`，避免真实 ID 造成高基数。

## 业务自定义指标

```go
requests := metricsSvc.NewCounter("orders_total", "订单总数", "status")
requests.WithLabelValues("paid").Inc()

connections := metricsSvc.NewGauge("db_connections", "数据库连接数")
connections.WithLabelValues().Set(12)

latency := metricsSvc.NewHistogram("api_latency_seconds", "接口延迟")
latency.WithLabelValues().Observe(0.15)
```

同名指标复用已注册实例，重复调用不 panic。

## 配置

```yaml
metrics:
  path: /metrics    # 采集端点路径，默认 /metrics
  namespace: myapp  # 指标名前缀，默认空
```

| 配置项 | 类型 | 说明 |
|--------|------|------|
| `path` | string | 采集端点路径，默认 `/metrics` |
| `namespace` | string | 指标名前缀，默认空 |

## 测试与 mock

业务单测中 mock 掉指标服务，见 `mocks/`：

```go
import "github.com/hecc-blot/metrics/mocks"

mockMetrics := &mocks.MockMetrics{}
```

## 相关模块

| 模块 | 说明 |
|------|------|
| [framework](https://github.com/hecc-blot/framework) | `iCoreApi.IMiddleware` 中间件契约 |
