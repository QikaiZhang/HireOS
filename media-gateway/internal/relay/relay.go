// Package relay 实现会话级媒体中继的核心逻辑：
//
//	浏览器 WS ──Offer──▶ [inbound 有界缓冲] ──Consume──▶ worker (gRPC)
//	浏览器 WS ◀──Outbound◀ [outbound 有界缓冲] ◀Push── worker (TTS)
//
// 实时音频"宁丢不等"：缓冲达到水位线时丢最旧帧，读写两侧永不阻塞。
// 本包不依赖 WebSocket / gRPC / protobuf，是纯逻辑，重点单测对象。
package relay

import (
	"context"
	"sync"
	"sync/atomic"
)

// DefaultCodec v1 音频编码标签（网关不解码，仅透传给下游消费者）
const DefaultCodec = "pcm_s16le_16k_mono"

// AudioFrame 一帧媒体数据。音频 40ms PCM ≈ 1280 字节。
type AudioFrame struct {
	RoomID      string
	Seq         uint32
	TimestampMs int64
	Codec       string
	Data        []byte
	Final       bool
}

// OutboundKind 下行消息类型：audio 走 WS 二进制，text/ended 走 WS 文本(JSON)
type OutboundKind int

const (
	OutboundAudio OutboundKind = iota
	OutboundText
	OutboundEnded
)

// OutboundMessage 下行消息（TTS 音频 / 字幕 / 结束通知）
type OutboundMessage struct {
	Kind OutboundKind
	Data []byte // audio: 媒体字节；text/ended: JSON
}

// DropReason 丢帧原因，用于指标与排障
type DropReason string

const (
	DropWatermark    DropReason = "watermark"     // 缓冲过水位，丢最旧
	DropSessionEnded DropReason = "session_ended" // 会话已关闭，到达的帧被拒
)

// Session 一个房间的中继会话。并发安全：读写泵、gRPC 流、控制协程共用。
type Session struct {
	RoomID string

	inbound  chan AudioFrame      // 浏览器 → worker
	outbound chan OutboundMessage // worker → 浏览器

	capacity   int
	watermark  int // 达到此长度即触发丢最旧（默认 capacity*80%）
	droppedIn  atomic.Uint64
	droppedOut atomic.Uint64

	nextSeq atomic.Uint32 // 网关侧分配的上行 seq（M1 由网关生成，浏览器只发字节）

	closed     atomic.Bool
	closeOnce  sync.Once
	done       chan struct{}
	closeCause atomic.Value // string

	browserAttached atomic.Bool // 一个房间同时只允许一个浏览器连接（worker 先接入属正常时序）
}

// ClaimBrowser 标记浏览器端已接入本会话；已被占用返回 false。
// worker 先于浏览器创建会话（正常时序），浏览器接入时必须能成功认领。
func (s *Session) ClaimBrowser() bool {
	return s.browserAttached.CompareAndSwap(false, true)
}

// NewSession 创建会话。capacity 为每侧缓冲帧数（默认 50 ≈ 2s 音频量）。
func NewSession(roomID string, capacity int) *Session {
	if capacity <= 0 {
		capacity = 50
	}
	s := &Session{
		RoomID:    roomID,
		inbound:   make(chan AudioFrame, capacity),
		outbound:  make(chan OutboundMessage, capacity),
		capacity:  capacity,
		watermark: capacity * 4 / 5, // 80% 水位
		done:      make(chan struct{}),
	}
	s.closeCause.Store("")
	return s
}

// Offer 浏览器上行帧入缓冲（非阻塞，水位丢帧，永不阻塞读泵）。
func (s *Session) Offer(f AudioFrame) {
	if s.closed.Load() {
		s.droppedIn.Add(1)
		return
	}
	if f.Seq == 0 {
		f.Seq = s.nextSeq.Add(1)
	}

	// 超水位：丢最旧一帧，给新帧腾位置。实时流中陈旧帧已失去播放价值。
	if len(s.inbound) >= s.watermark {
		s.dropOldestInbound(DropWatermark)
	}

	select {
	case s.inbound <- f:
	default:
		// 水位检查与 send 之间的竞态兜底：再丢一帧最旧后重试一次
		s.dropOldestInbound(DropWatermark)
		select {
		case s.inbound <- f:
		default:
			s.droppedIn.Add(1)
		}
	}
}

// Consume worker 侧取上行帧；会话关闭且缓冲取空后返回 false。
func (s *Session) Consume(ctx context.Context) (AudioFrame, bool) {
	for {
		select {
		case f := <-s.inbound:
			return f, true
		case <-ctx.Done():
			return AudioFrame{}, false
		case <-s.done:
			// 关闭后先排干存量，避免丢掉已缓冲的有效音频
			select {
			case f := <-s.inbound:
				return f, true
			default:
				return AudioFrame{}, false
			}
		}
	}
}

// PushOutbound worker 下行消息入缓冲（非阻塞，同水位丢帧策略）。
func (s *Session) PushOutbound(m OutboundMessage) {
	if s.closed.Load() {
		s.droppedOut.Add(1)
		return
	}
	if len(s.outbound) >= s.watermark {
		s.dropOldestOutbound(DropWatermark)
	}
	select {
	case s.outbound <- m:
	default:
		s.dropOldestOutbound(DropWatermark)
		select {
		case s.outbound <- m:
		default:
			s.droppedOut.Add(1)
		}
	}
}

// PopOutbound 写泵取下行消息；会话关闭且缓冲取空后返回 false。
func (s *Session) PopOutbound(ctx context.Context) (OutboundMessage, bool) {
	for {
		select {
		case m := <-s.outbound:
			return m, true
		case <-ctx.Done():
			return OutboundMessage{}, false
		case <-s.done:
			select {
			case m := <-s.outbound:
				return m, true
			default:
				return OutboundMessage{}, false
			}
		}
	}
}

// Close 关闭会话，cause 记录关闭原因（normal/drain/browser_closed/...）。
// 关闭后两侧缓冲中的存量消息仍可被 Consume/PopOutbound 排干。
func (s *Session) Close(cause string) {
	s.closeOnce.Do(func() {
		s.closeCause.Store(cause)
		s.closed.Store(true)
		close(s.done)
	})
}

// Closed 返回会话是否已关闭及关闭原因。
func (s *Session) Closed() (bool, string) {
	return s.closed.Load(), s.closeCause.Load().(string)
}

// DroppedInbound / DroppedOutbound 丢帧计数（测试与指标用）
func (s *Session) DroppedInbound() uint64  { return s.droppedIn.Load() }
func (s *Session) DroppedOutbound() uint64 { return s.droppedOut.Load() }

// LenInbound 当前上行缓冲长度（指标与测试用）
func (s *Session) LenInbound() int { return len(s.inbound) }

func (s *Session) dropOldestInbound(reason DropReason) {
	select {
	case <-s.inbound:
		s.droppedIn.Add(1)
		onDrop(reason)
	default:
	}
}

func (s *Session) dropOldestOutbound(reason DropReason) {
	select {
	case <-s.outbound:
		s.droppedOut.Add(1)
		onDrop(reason)
	default:
	}
}

// onDrop 指标挂钩，由 metrics 包注入（避免 relay 直接依赖具体指标实现）
var onDrop = func(DropReason) {}

// SetDropHook 供 main 装配指标回调，单测环境不装配即为 noop
func SetDropHook(fn func(DropReason)) { onDrop = fn }
