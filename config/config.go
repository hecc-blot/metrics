package config

// Config 监控指标配置。
type Config struct {
	// Path Prometheus 采集端点路径，默认 "/metrics"。
	Path string `mapstructure:"path"`
	// Namespace 指标名前缀（如 "myapp"），所有业务指标与进程指标统一加前缀，默认空。
	Namespace string `mapstructure:"namespace"`
}

// Normalize 补全默认值，供构造指标服务时调用。
func Normalize(cfg Config) Config {
	if cfg.Path == "" {
		cfg.Path = "/metrics"
	}
	return cfg
}
