package main

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"media-gateway/gen/mediav1"
	"media-gateway/internal/config"
	"media-gateway/internal/metrics"
	"media-gateway/internal/registry"
	"media-gateway/internal/relay"
	"media-gateway/internal/rpc"
	"media-gateway/internal/server"
)

//go:embed echo.html
var echoPage []byte

func main() {
	cfg := config.Default()
	cfg.BindFlags(flag.CommandLine)
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	// relay 丢帧挂钩 → Prometheus
	relay.SetDropHook(func(reason relay.DropReason) {
		metrics.FramesDroppedTotal.WithLabelValues(string(reason)).Inc()
	})

	reg := registry.NewLocal()

	// gRPC：MediaRelay 服务
	grpcSrv := grpc.NewServer(grpc.KeepaliveParams(keepalive.ServerParameters{
		Time:    30 * time.Second,
		Timeout: 10 * time.Second,
	}))
	mediav1.RegisterMediaRelayServer(grpcSrv, &rpc.RelayServer{Registry: reg, Logger: log})

	// HTTP：WS + 健康检查 + 指标 + 回声测试页
	mux := http.NewServeMux()
	mux.Handle("/ws", server.NewWSHandler(cfg, reg, log))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		// M1： readiness = 进程可用；M2 增加 Redis 可达性检查
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/echo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(echoPage)
	})

	httpSrv := &http.Server{Addr: cfg.HTTPAddr, Handler: mux}

	// 优雅退出：停收新连接 → 关 HTTP/gRPC → 遗留会话随进程终止（M2 由 drain 流程接管）
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("HTTP 服务异常退出", "err", err)
			os.Exit(1)
		}
	}()
	ln, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		log.Error("gRPC 监听失败", "addr", cfg.GRPCAddr, "err", err)
		os.Exit(1)
	}
	go func() {
		if err := grpcSrv.Serve(ln); err != nil {
			log.Error("gRPC 服务异常退出", "err", err)
		}
	}()

	log.Info("media-gateway 已启动",
		"http", cfg.HTTPAddr, "grpc", cfg.GRPCAddr,
		"session_buffer", cfg.SessionBuffer, "max_conns", cfg.MaxConnections)
	log.Info("回声测试页", "url", "http://localhost"+cfg.HTTPAddr+"/echo?room_id=demo-1")

	<-ctx.Done()
	log.Info("收到退出信号，开始优雅关闭")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	grpcSrv.GracefulStop()
	log.Info("media-gateway 已退出")
}
