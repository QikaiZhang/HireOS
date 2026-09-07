package relay

import (
	"context"
	"sync"
	"testing"
	"time"
)

func makeFrame(seq uint32, size int) AudioFrame {
	return AudioFrame{RoomID: "r1", Seq: seq, Data: make([]byte, size), Codec: DefaultCodec}
}

// 水位丢帧：灌满后继续灌，缓冲不超过容量，最旧帧被丢弃，计数正确
func TestOfferWatermarkDropOldest(t *testing.T) {
	s := NewSession("r1", 10)
	defer s.Close("test")

	for i := 0; i < 100; i++ {
		s.Offer(makeFrame(0, 16)) // seq=0 → 网关分配 1..100
	}

	// 稳态长度 = 水位线（capacity 10 × 80% = 8）：超过水位即丢最旧
	if got := len(s.inbound); got != 8 {
		t.Fatalf("缓冲长度 = %d, want 8（水位线）", got)
	}
	if s.DroppedInbound() == 0 {
		t.Fatal("超量灌入后应产生丢帧计数")
	}

	// 最旧被丢弃：现存第一帧的 seq 应大于 1
	first, ok := s.Consume(context.Background())
	if !ok {
		t.Fatal("缓冲应有存量帧")
	}
	if first.Seq <= 1 {
		t.Fatalf("最旧帧应已被丢弃，现存首帧 seq = %d", first.Seq)
	}
}

// 正常消费不丢帧：生产 5 帧（低于水位），全部可按序取出
func TestNoDropBelowWatermark(t *testing.T) {
	s := NewSession("r1", 10)
	defer s.Close("test")

	for i := 0; i < 5; i++ {
		s.Offer(makeFrame(0, 16))
	}
	if s.DroppedInbound() != 0 {
		t.Fatalf("低于水位不应丢帧，dropped = %d", s.DroppedInbound())
	}
	for want := uint32(1); want <= 5; want++ {
		f, ok := s.Consume(context.Background())
		if !ok {
			t.Fatalf("第 %d 帧缺失", want)
		}
		if f.Seq != want {
			t.Fatalf("seq = %d, want %d（必须按序）", f.Seq, want)
		}
	}
}

// 会话关闭后：Consume 排干存量后返回 false，Offer 直接拒绝
func TestCloseDrainsThenStops(t *testing.T) {
	s := NewSession("r1", 10)
	for i := 0; i < 3; i++ {
		s.Offer(makeFrame(0, 16))
	}
	s.Close("normal")

	// 关闭后新帧被拒
	s.Offer(makeFrame(0, 16))
	if got := s.DroppedInbound(); got != 1 {
		t.Fatalf("关闭后到达帧应计入丢弃，dropped = %d", got)
	}

	for i := 0; i < 3; i++ {
		if _, ok := s.Consume(context.Background()); !ok {
			t.Fatalf("关闭后应先排干存量，第 %d 帧失败", i)
		}
	}
	if _, ok := s.Consume(context.Background()); ok {
		t.Fatal("存量排干后 Consume 应返回 false")
	}
}

// Consume 在无数据时阻塞，Close 能唤醒
func TestConsumeUnblockedByClose(t *testing.T) {
	s := NewSession("r1", 10)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if _, ok := s.Consume(context.Background()); ok {
			t.Error("Close 后 Consume 应返回 false")
		}
	}()

	time.Sleep(20 * time.Millisecond)
	s.Close("browser_closed")
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close 未唤醒阻塞的 Consume")
	}
}

// 下行通道同等策略
func TestOutboundWatermarkDrop(t *testing.T) {
	s := NewSession("r1", 10)
	defer s.Close("test")

	for i := 0; i < 50; i++ {
		s.PushOutbound(OutboundMessage{Kind: OutboundAudio, Data: make([]byte, 16)})
	}
	if got := len(s.outbound); got != 8 {
		t.Fatalf("下行缓冲长度 = %d, want 8（水位线）", got)
	}
	if s.DroppedOutbound() == 0 {
		t.Fatal("下行超量应产生丢帧计数")
	}
}

// 浏览器端认领：首次成功，重复失败（一个房间只允许一个浏览器连接）
func TestClaimBrowser(t *testing.T) {
	s := NewSession("r1", 10)
	defer s.Close("test")

	if !s.ClaimBrowser() {
		t.Fatal("首次认领应成功")
	}
	if s.ClaimBrowser() {
		t.Fatal("重复认领应失败")
	}
}

// 并发灌入不越界、总数守恒（有界 + 计数），用于暴露 data race（-race 下运行）
func TestConcurrentOffer(t *testing.T) {
	s := NewSession("r1", 32)

	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				s.Offer(makeFrame(0, 16))
			}
		}()
	}
	wg.Wait()

	if got := len(s.inbound); got > s.capacity {
		t.Fatalf("缓冲越界：len = %d, capacity = %d", got, s.capacity)
	}
	// 先关闭再排干：Consume 只在会话关闭且缓冲取空后才返回 false
	s.Close("test")
	consumed := 0
	for {
		if _, ok := s.Consume(context.Background()); !ok {
			break
		}
		consumed++
	}
	total := uint64(consumed) + s.DroppedInbound()
	if total != 8*200 {
		t.Fatalf("帧数守恒失败：consumed=%d dropped=%d want total=%d", consumed, s.DroppedInbound(), 8*200)
	}
}
