// Package server 实现浏览器侧 WebSocket 接入：
// 读泵收音频字节 → relay 会话缓冲；写泵取下行消息 → 浏览器。
// 客户端协议（M1）：
//   - 连接：/ws?room_id=xxx&codec=pcm_s16le_16k_mono
//   - 上行二进制帧 = 原始音频字节（40ms PCM ≈ 1280B），seq/timestamp 由网关分配
//   - 下行二进制帧 = TTS 音频字节；下行文本帧 = JSON（text / session_ended）
package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"media-gateway/internal/config"
	"media-gateway/internal/metrics"
	"media-gateway/internal/registry"
	"media-gateway/internal/relay"
)

type WSServer struct {
	Cfg      *config.Config
	Registry registry.Registry
	Logger   *slog.Logger

	conns atomic.Int64
}

func NewWSHandler(cfg *config.Config, reg registry.Registry, log *slog.Logger) *WSServer {
	return &WSServer{Cfg: cfg, Registry: reg, Logger: log}
}

var upgrader = websocket.AcceptOptions{
	// M1 本地联调放开同源限制；M2 接入认证体系时收敛为 Origin 白名单
	InsecureSkipVerify: true,
}

func (h *WSServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	roomID := r.URL.Query().Get("room_id")
	if roomID == "" {
		http.Error(w, "缺少 room_id", http.StatusBadRequest)
		return
	}
	codec := r.URL.Query().Get("codec")
	if codec == "" {
		codec = relay.DefaultCodec
	}

	// 连接数硬顶：超限快速失败，防止资源耗尽
	if h.conns.Load() >= int64(h.Cfg.MaxConnections) {
		metrics.WSConnectionsRejectedTotal.Inc()
		http.Error(w, "节点连接数已达上限", http.StatusServiceUnavailable)
		return
	}

	socket, err := websocket.Accept(w, r, &upgrader)
	if err != nil {
		return
	}
	metrics.WSConnectionsTotal.Inc()
	h.serveConn(r.Context(), socket, roomID, codec)
}

func (h *WSServer) serveConn(reqCtx context.Context, socket *websocket.Conn, roomID, codec string) {
	log := h.Logger.With("room_id", roomID)
	defer socket.Close(websocket.StatusInternalError, "内部错误")

	// 接入会话：worker 先创建的活跃会话可被浏览器认领；已关闭的则重建；
	// 浏览器端已被占用（真正的重复接入）才拒绝。
	sess, ok := h.Registry.Lookup(roomID)
	if ok {
		if closed, _ := sess.Closed(); closed {
			h.Registry.Unbind(roomID)
			sess = nil
		} else if !sess.ClaimBrowser() {
			log.Warn("拒绝重复房间连接")
			socket.Close(websocket.StatusPolicyViolation, "房间已有活跃连接")
			return
		}
	}
	if sess == nil {
		sess = relay.NewSession(roomID, h.Cfg.SessionBuffer)
		if err := h.Registry.Bind(roomID, sess); err != nil {
			log.Warn("绑定会话失败", "err", err)
			socket.Close(websocket.StatusPolicyViolation, "房间绑定失败")
			return
		}
	}
	metrics.SessionsActive.Inc()
	h.conns.Add(1)

	// 连接级 context：读泵退出或请求终止时取消，写泵/心跳随之收尾
	ctx, cancel := context.WithCancel(reqCtx)
	defer cancel()
	done := make(chan struct{})
	go h.writePump(ctx, socket, sess, done)
	h.readPump(ctx, socket, sess, codec)

	// 读泵退出即浏览器侧断开：关闭会话，gRPC 上行泵随之向 worker 发终止通知
	sess.Close("browser_closed")
	metrics.SessionsActive.Dec()
	h.conns.Add(-1)
	h.Registry.Unbind(roomID)
	<-done
	log.Info("连接结束", "dropped_in", sess.DroppedInbound(), "dropped_out", sess.DroppedOutbound())
}

// readPump 浏览器上行：二进制=音频字节；文本=JSON 控制（{"type":"stop"}）。
// 读超时：首帧 FirstFrameSec，之后每帧 IdleTimeout —— 实时面试中持续静音超过
// IdleTimeout 等价于死连接，直接回收。
func (h *WSServer) readPump(ctx context.Context, socket *websocket.Conn, sess *relay.Session, codec string) {
	socket.SetReadLimit(h.Cfg.MaxFrameBytes)

	firstFrame := true
	for {
		deadline := h.Cfg.IdleTimeout
		if firstFrame {
			deadline = h.Cfg.FirstFrameSec
		}
		readCtx, cancelRead := context.WithTimeout(ctx, deadline)
		msgType, data, err := socket.Read(readCtx)
		cancelRead()
		if err != nil {
			return
		}
		firstFrame = false

		switch msgType {
		case websocket.MessageBinary:
			metrics.FramesInTotal.Inc()
			sess.Offer(relay.AudioFrame{
				RoomID:      sess.RoomID,
				TimestampMs: time.Now().UnixMilli(),
				Codec:       codec,
				Data:        data,
			})
		case websocket.MessageText:
			var ctl struct {
				Type string `json:"type"`
			}
			if json.Unmarshal(data, &ctl) == nil && ctl.Type == "stop" {
				sess.Close("normal")
				return
			}
		}
	}
}

// writePump 网关下行：audio=二进制；text/ended=JSON 文本。写超时即断开（慢消费者保护）。
func (h *WSServer) writePump(ctx context.Context, socket *websocket.Conn, sess *relay.Session, done chan struct{}) {
	defer close(done)
	go pingLoop(ctx, socket, h.Cfg.PingInterval)

	for {
		msg, ok := sess.PopOutbound(ctx)
		if !ok {
			// 会话关闭且缓冲排干：正常收尾
			_ = socket.Close(websocket.StatusNormalClosure, "")
			return
		}

		writeCtx, cancelWrite := context.WithTimeout(ctx, h.Cfg.WriteTimeout)
		var err error
		switch msg.Kind {
		case relay.OutboundAudio:
			metrics.FramesOutTotal.Inc()
			err = socket.Write(writeCtx, websocket.MessageBinary, msg.Data)
		case relay.OutboundText:
			err = socket.Write(writeCtx, websocket.MessageText, msg.Data)
		case relay.OutboundEnded:
			_ = socket.Write(writeCtx, websocket.MessageText, msg.Data)
			cancelWrite()
			_ = socket.Close(websocket.StatusNormalClosure, "会话结束")
			return
		}
		cancelWrite()
		if err != nil {
			metrics.WSWriteTimeoutsTotal.Inc()
			sess.Close("error")
			return
		}
	}
}

// pingLoop 主动心跳：探测半开连接，浏览器会自动回 pong
func pingLoop(ctx context.Context, socket *websocket.Conn, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = socket.Ping(ctx)
		}
	}
}
