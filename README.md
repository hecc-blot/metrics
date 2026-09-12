# hecc-blot-metrics

监控指标模块（Prometheus）：Counter / Gauge / Histogram、采集端点、HTTP 请求指标中间件。

## 安装

```bash
go get github.com/hecc-blot/metrics
```

## 快速开始

```go
import (
	"net/http"

	metricsContract "github.com/hecc-blot/metrics/contract"
	metricsConfig "github.com/hecc-blot/metrics/config"
	metrics "github.com/hecc-blot/metrics/service"
)

func main() {
	// 1. 构造（无 cleanup、无 error）
	cfg := metricsConfig.Normalize(config.Metrics)
	metricsSvc := metrics.NewMetrics(&cfg)

	// 2. 注册到容器（供业务注入）
	container.Set(new(metricsContract.IMetrics), metricsSvc)

	// 3. 挂载采集端点与中间件
	apiHandle.Handle(http.MethodGet, cfg.Path, metricsSvc.Handler())
	apiHandle.Middleware(metrics.NewHttpMiddleware(metricsSvc))
}
```

完整的应用组装顺序见[组织主页](https://github.com/hecc-blot)。

## 接口定义

```go
import metricsContract "github.com/hecc-blot/metrics/contract"

type IMetrics interface {
	// Handler 返回 Prometheus 采集端点，供挂载到 HTTP 路由
	Handler() http.Handler

	// 创建带标签指标。同名指标复用已注册实例，重复调用不 panic。
	NewCounter(name, help string, labelNames ...string) ICounterVec
	NewGauge(name, help string, labelNames ...string) IGaugeVec
	NewHistogram(name, help string, labelNames ...string) IHistogramVec
}

type ICounter interface {
	Inc()
	Add(delta float64)
}

type IGauge interface {
	Inc()
	Dec()
	Add(delta float64)
	Set(value float64)
}

type IHistogram interface {
	Observe(value float64)
}

// 三种 Vec 都有 WithLabelValues(lvs ...string)，按标签值定位到具体指标
type ICounterVec interface{ WithLabelValues(lvs ...string) ICounter }
type IGaugeVec interface{ WithLabelValues(lvs ...string) IGauge }
type IHistogramVec interface{ WithLabelValues(lvs ...string) IHistogram }
```

## 使用

### 挂载端点与中间件

```go
// 采集端点：用 Handle 走原生 http.Handler，绕过参数绑定与响应包装
apiHandle.Handle(http.MethodGet, cfg.Path, metricsSvc.Handler())

// 请求指标中间件：自动记录 QPS / 延迟 / 错误率
apiHandle.Middleware(metrics.NewHttpMiddleware(metricsSvc))
```

中间件内置的指标：

| 指标名（含 namespace 前缀） | 类型 | 标签 |
|---|---|---|
| `http_requests_total` | Counter | `method` / `path` / `status` |
| `http_request_duration_seconds` | Histogram | `method` / `path` / `status` |

> `path` 标签取的是**路由模板**（如 `/account/:id`），不是原始 URL —— 避免高基数。未匹配到路由时回退为 `unmatched`。

### 业务自定义指标

```go
type OrderApi struct {
	Metrics metricsContract.IMetrics `inject:""`
}

// 注册期一次性创建（同名复用，不会重复注册）
var orderCounter metricsContract.ICounterVec

func (a *OrderApi) Call(ctx iCoreApi.IContext) (interface{}, coreError.IError) {
	if orderCounter == nil {
		orderCounter = a.Metrics.NewCounter("orders_total", "订单总数", "status")
	}
	orderCounter.WithLabelValues("success").Inc()
	// ...
}
```

三种指标类型：

```go
counter := metricsSvc.NewCounter("orders_total", "订单总数", "status")
counter.WithLabelValues("success").Inc()
counter.WithLabelValues("fail").Add(1)

gauge := metricsSvc.NewGauge("queue_length", "队列长度")
gauge.Set(42)
gauge.Inc()

histogram := metricsSvc.NewHistogram("order_amount", "订单金额", "channel")
histogram.WithLabelValues("web").Observe(99.5)
```

## 配置

```yaml
metrics:
  path: /metrics        # 采集端点路径，默认 /metrics
  namespace: myapp      # 指标名前缀，默认空
```

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `path` | string | `/metrics` | 采集端点路径 |
| `namespace` | string | 空 | 指标名前缀 |

> ⚠️ **`path` 不会从 `IMetrics` 读回来**。`NewMetrics` 内部只用 `Namespace`，`Path` 被丢弃，`IMetrics` 也没有 Path 访问器。所以**业务必须自己持有这个值**（上面示例里用 `cfg.Path`），传 `nil` 给 `NewMetrics` 并不会让框架帮你带出默认端点。

### namespace 的作用范围

`Namespace` 会加在**业务指标**与**进程指标**上，但**不加在 Go 运行时指标**上：

| 指标 | 是否带前缀 |
|---|---|
| 业务指标（`NewCounter` 等） | ✅ `myapp_orders_total` |
| 进程指标 `process_*` | ✅ `myapp_process_cpu_seconds_total` |
| Go 运行时指标 `go_*` | ❌ `go_goroutines`（`NewGoCollector` 不接受 namespace） |

## 语义与约定

- **同名指标复用**：`NewCounter` / `NewGauge` / `NewHistogram` 传入同名时返回已注册的实例，不会 panic，也不会重复注册。
- **Histogram 不支持自定义 buckets**：只传 Name / Help / Labels，使用 Prometheus 客户端库的默认桶。延迟类指标想看 P95/P99 只能落在默认桶边界上。
- **中间件与端点必须挂上**：不 `Middleware(NewHttpMiddleware(...))` 就没有 HTTP 指标；不 `Handle(path, Handler())` 就采集不到任何数据。
- **并发安全**：`IMetrics` 与各 Vec 实例可跨 goroutine 共享（内部加锁）。
- **无释放语义**：纯内存注册表，不需要 cleanup。
- **`NewMetrics(nil)` 可用**：等价于零值配置（namespace 为空）。

## 测试与 mock

```go
import "github.com/hecc-blot/metrics/mocks"

mock := &mocks.MockMetrics{} // 各方法通过 Fn 字段定制
```

## 相关模块

| 模块 | 说明 |
|------|------|
| [framework](https://github.com/hecc-blot/framework) | `Handle` 挂载端点、`IMiddleware` 挂载中间件 |
| [sse](https://github.com/hecc-blot/sse) | `Stats()` 可暴露为连接数指标 |
| [trace](https://github.com/hecc-blot/trace) | 与指标同为可观测能力，常一起挂载 |
