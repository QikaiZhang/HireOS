// Package rpc 实现 MediaRelay gRPC 服务端：
// 网关把浏览器上行音频灌给 worker，把 worker 下行的 TTS/文本推给浏览器。
package rpc

import (
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"media-gateway/gen/mediav1"
	"media-gateway/internal/metrics"
	"media-gateway/internal/registry"
	"media-gateway/internal/relay"
)

type RelayServer struct {
	mediav1.UnimplementedMediaRelayServer

	Registry registry.Registry
	Logger   *slog.Logger
}

// SessionStream 房间级双向流。worker 为 client，首帧必须是 start/resume 控制帧。
func (s *RelayServer) SessionStream(stream mediav1.MediaRelay_SessionStreamServer) error {
	log := s.Logger

	first, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "首帧读取失败: %v", err)
	}
	ctl := first.GetControl()
	if ctl == nil {
		return status.Error(codes.InvalidArgument, "首帧必须是 StreamControl(start/resume)")
	}
	if ctl.Event != "start" && ctl.Event != "resume" {
		return status.Errorf(codes.InvalidArgument, "不支持的控制事件: %q", ctl.Event)
	}

	roomID := ctl.RoomId
	if roomID == "" {
		return status.Error(codes.InvalidArgument, "room_id 不能为空")
	}

	// worker 先于浏览器接入时由 worker 创建会话；已存在则复用（resume 语义）
	sess, ok := s.Registry.Lookup(roomID)
	if !ok {
		sess = relay.NewSession(roomID, defaultSessionBuffer)
		if err := s.Registry.Bind(roomID, sess); err != nil {
			return status.Errorf(codes.AlreadyExists, "绑定会话失败: %v", err)
		}
	}
	log = log.With("room_id", roomID)
	log.Info("worker 流已接入", "event", ctl.Event, "token_len", len(ctl.Token))

	// 上行泵：会话缓冲 → worker。流结束或会话关闭后向 worker 推终止通知。
	upstreamDone := make(chan struct{})
	go func() {
		defer close(upstreamDone)
		s.pumpUpstream(stream, sess, log)
	}()

	// 下行主循环：worker 帧 → 会话下行缓冲
	for {
		wf, err := stream.Recv()
		if err != nil {
			// worker 断流：会话保留（浏览器侧缓冲继续），等待 resume 重挂（M1 记录，M2 加 grace TTL）
			log.Warn("worker 流断开，会话保留等待重挂", "err", err)
			metrics.GRPCStreamErrorsTotal.WithLabelValues("broken").Inc()
			return nil
		}

		switch p := wf.Payload.(type) {
		case *mediav1.WorkerFrame_TtsAudio:
			a := p.TtsAudio
			sess.PushOutbound(relay.OutboundMessage{Kind: relay.OutboundAudio, Data: a.Data})
		case *mediav1.WorkerFrame_Text:
			sess.PushOutbound(relay.OutboundMessage{
				Kind: relay.OutboundText,
				Data: jsonText(p.Text.Role, p.Text.Text),
			})
		case *mediav1.WorkerFrame_Ended:
			log.Info("worker 请求结束会话", "reason", p.Ended.Reason)
			sess.PushOutbound(relay.OutboundMessage{
				Kind: relay.OutboundEnded,
				Data: jsonEnded(p.Ended.Reason),
			})
			sess.Close("normal")
			<-upstreamDone
			return nil
		case *mediav1.WorkerFrame_Control:
			if p.Control.Event == "stop" {
				sess.Close("normal")
				<-upstreamDone
				return nil
			}
			log.Warn("忽略流中控制帧", "event", p.Control.Event)
		}
	}
}

// pumpUpstream 把会话上行帧经 gRPC 推给 worker，并统计转发延迟
func (s *RelayServer) pumpUpstream(
	stream mediav1.MediaRelay_SessionStreamServer,
	sess *relay.Session,
	log *slog.Logger,
) {
	for {
		frame, ok := sess.Consume(stream.Context())
		if !ok {
			// 会话关闭（浏览器断开/drain/normal）→ 通知 worker 终止原因
			_, cause := sess.Closed()
			if err := stream.Send(&mediav1.GatewayFrame{Payload: &mediav1.GatewayFrame_Ended{
				Ended: &mediav1.SessionEnded{RoomId: sess.RoomID, Reason: cause},
			}}); err != nil {
				log.Warn("终止通知发送失败", "err", err)
			}
			return
		}

		sendStart := time.Now()
		err := stream.Send(&mediav1.GatewayFrame{Payload: &mediav1.GatewayFrame_Audio{
			Audio: &mediav1.AudioChunk{
				RoomId:      frame.RoomID,
				Seq:         frame.Seq,
				TimestampMs: frame.TimestampMs,
				Codec:       frame.Codec,
				Data:        frame.Data,
				Final:       frame.Final,
			},
		}})
		if err != nil {
			log.Warn("上行帧发送失败", "seq", frame.Seq, "err", err)
			metrics.GRPCStreamErrorsTotal.WithLabelValues("send_failed").Inc()
			return
		}
		// 转发延迟 = 帧离开网关时刻 - 帧到达网关时刻
		metrics.RelayLatency.Observe(sendStart.Sub(time.UnixMilli(frame.TimestampMs)).Seconds())
	}
}

// M1 简化：worker 与浏览器接入顺序不定，worker 先到时用默认缓冲；M2 统一由
// registry 管理会话生命周期与 TTL。
const defaultSessionBuffer = 50

func jsonText(role, text string) []byte {
	return fmt.Appendf(nil, `{"type":"text","role":%q,"text":%q}`, role, text)
}

func jsonEnded(reason string) []byte {
	return fmt.Appendf(nil, `{"type":"session_ended","reason":%q}`, reason)
}
