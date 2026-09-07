// Package config 网关配置。全部有默认值，可用 flag 覆盖（容器环境用 env 注入 flag）。
// 限额默认值对应 GATEWAY_SPEC「容量与资源模型」的 v1 验收门。
package config

import (
	"flag"
	"time"
)

type Config struct {
	HTTPAddr string // WS/健康检查/指标监听地址
	GRPCAddr string // MediaRelay gRPC 监听地址

	SessionBuffer   int           // 每会话每侧缓冲帧数（50 帧 ≈ 2s PCM 音频）
	MaxFrameBytes   int64         // 单帧上限（资源保护）
	MaxConnections  int           // 节点最大连接数硬顶
	IdleTimeout     time.Duration // WS 空闲超时（pong 刷新）
	PingInterval    time.Duration // 网关心跳间隔
	WriteTimeout    time.Duration // 下行单次写超时（超时断开，防慢消费者拖垮）
	FirstFrameSec   time.Duration // 握手后首帧超时
	ShutdownTimeout time.Duration // 优雅退出总时长
}

func Default() *Config {
	return &Config{
		HTTPAddr:        ":8080",
		GRPCAddr:        ":9090",
		SessionBuffer:   50,
		MaxFrameBytes:   8192,
		MaxConnections:  8000,
		IdleTimeout:     30 * time.Second,
		PingInterval:    20 * time.Second,
		WriteTimeout:    5 * time.Second,
		FirstFrameSec:   10 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}
}

// BindFlags 将配置注册到 flag 集（main 中调用）
func (c *Config) BindFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.HTTPAddr, "http-addr", c.HTTPAddr, "HTTP/WS 监听地址")
	fs.StringVar(&c.GRPCAddr, "grpc-addr", c.GRPCAddr, "gRPC 监听地址")
	fs.IntVar(&c.SessionBuffer, "session-buffer", c.SessionBuffer, "每会话缓冲帧数")
	fs.Int64Var(&c.MaxFrameBytes, "max-frame-bytes", c.MaxFrameBytes, "单帧字节数上限")
	fs.IntVar(&c.MaxConnections, "max-connections", c.MaxConnections, "节点最大连接数")
	fs.DurationVar(&c.IdleTimeout, "idle-timeout", c.IdleTimeout, "WS 空闲超时")
	fs.DurationVar(&c.WriteTimeout, "write-timeout", c.WriteTimeout, "下行写超时")
	fs.DurationVar(&c.ShutdownTimeout, "shutdown-timeout", c.ShutdownTimeout, "优雅退出时长")
}
