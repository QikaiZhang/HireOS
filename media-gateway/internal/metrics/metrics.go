// Package metrics 定义网关的 Prometheus 指标。
// 命名规范见 GATEWAY_SPEC「可观测性」，label 只用低基数字段，禁止 room_id。
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SessionsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gateway_sessions_active",
		Help: "当前活跃会话数",
	})

	FramesInTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_frames_in_total",
		Help: "浏览器上行音频帧数",
	})

	FramesOutTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_frames_out_total",
		Help: "推送给浏览器的下行音频帧数",
	})

	FramesDroppedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_frames_dropped_total",
		Help: "丢帧数（按原因）",
	}, []string{"reason"})

	RelayLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gateway_relay_latency_seconds",
		Help:    "帧从 WS 入口到 gRPC 出口的转发延迟",
		Buckets: []float64{0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1},
	})

	WSWriteTimeoutsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_ws_write_timeouts_total",
		Help: "下行写超时次数",
	})

	GRPCStreamErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gateway_grpc_stream_errors_total",
		Help: "gRPC 流错误（按类型）",
	}, []string{"kind"})

	WSConnectionsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_ws_connections_total",
		Help: "累计 WS 连接数",
	})

	WSConnectionsRejectedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gateway_ws_connections_rejected_total",
		Help: "超限拒绝的 WS 连接数",
	})
)
