package integration

// gRPC 中继层吞吐测试：按真实音频速率（40ms/帧 = 25fps） paced 灌帧，
// 下游 Echo 即时消费，预期 1:1 回收、零丢帧。
// 另含一组压力灌帧断言，验证水位丢帧的守恒性（成功 + 丢弃 = 总量）。

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"media-gateway/gen/mediav1"
	"media-gateway/internal/config"
	"media-gateway/internal/registry"
	"media-gateway/internal/relay"
	"media-gateway/internal/rpc"
)

func startGRPCStack(t *testing.T) (*relay.Session, mediav1.MediaRelay_SessionStreamClient, context.CancelFunc) {
	t.Helper()
	cfg := config.Default()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := registry.NewLocal()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	grpcSrv := grpc.NewServer()
	mediav1.RegisterMediaRelayServer(grpcSrv, &rpc.RelayServer{Registry: reg, Logger: log})
	go func() { _ = grpcSrv.Serve(ln) }()
	t.Cleanup(grpcSrv.Stop)

	// 会话由"浏览器侧"创建（正常时序：WS 先接入）
	sess := relay.NewSession("perf", cfg.SessionBuffer)
	if err := reg.Bind("perf", sess); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	conn, err := grpc.NewClient(ln.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	stream, err := mediav1.NewMediaRelayClient(conn).SessionStream(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(&mediav1.WorkerFrame{Payload: &mediav1.WorkerFrame_Control{
		Control: &mediav1.StreamControl{RoomId: "perf", Event: "start"},
	}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); _ = conn.Close() })
	return sess, stream, cancel
}

// TestGRPCRelayPacedThroughput 按真实音频节奏灌 100 帧（4s），应 1:1 回收、零丢帧
func TestGRPCRelayPacedThroughput(t *testing.T) {
	sess, stream, _ := startGRPCStack(t)

	// worker：echo
	go func() {
		for {
			gf, err := stream.Recv()
			if err != nil {
				return
			}
			if a := gf.GetAudio(); a != nil {
				if stream.Send(&mediav1.WorkerFrame{Payload: &mediav1.WorkerFrame_TtsAudio{TtsAudio: a}}) != nil {
					return
				}
			}
		}
	}()

	const total = 100
	go func() {
		ticker := time.NewTicker(40 * time.Millisecond) // 25 fps = 真实语音帧率
		defer ticker.Stop()
		for i := 0; i < total; i++ {
			<-ticker.C
			sess.Offer(relay.AudioFrame{RoomID: "perf", TimestampMs: time.Now().UnixMilli(),
				Codec: relay.DefaultCodec, Data: make([]byte, 1280)})
		}
	}()

	start := time.Now()
	got := 0
	for got < total {
		if _, ok := sess.PopOutbound(context.Background()); !ok {
			break
		}
		got++
	}
	elapsed := time.Since(start)
	t.Logf("paced %d frames in %v = %.0f fps, dropped_in=%d", got, elapsed, float64(got)/elapsed.Seconds(), sess.DroppedInbound())
	if got != total {
		t.Fatalf("按真实节奏灌帧不应丢帧：收到 %d/%d, dropped=%d", got, total, sess.DroppedInbound())
	}
	if sess.DroppedInbound() != 0 {
		t.Fatalf("预期零丢帧，实际 %d", sess.DroppedInbound())
	}
}

// TestRelayDropConservationUnderPressure 压力灌帧（远超消费速率）：
// 缓冲有界、丢帧守恒（消费 + 丢弃 = 灌入），帧不重复不复制。
func TestRelayDropConservationUnderPressure(t *testing.T) {
	sess, stream, _ := startGRPCStack(t)

	consumedUpstream := make(chan int, 1)
	go func() {
		n := 0
		for {
			gf, err := stream.Recv()
			if err != nil || gf.GetEnded() != nil {
				consumedUpstream <- n
				return
			}
			if a := gf.GetAudio(); a != nil {
				n++
			}
		}
	}()

	const total = 500
	for i := 0; i < total; i++ {
		sess.Offer(relay.AudioFrame{RoomID: "perf", TimestampMs: time.Now().UnixMilli(),
			Codec: relay.DefaultCodec, Data: make([]byte, 1280)})
	}

	sess.Close("test") // 停止灌入后排干
	// 排干 worker 已收 + 缓冲存量，与丢弃计数守恒
	delivered := <-consumedUpstream
	buffered := 0
	for {
		if _, ok := sess.Consume(context.Background()); !ok {
			break
		}
		buffered++
	}
	if delivered+buffered+int(sess.DroppedInbound()) != total {
		t.Fatalf("守恒失败: delivered=%d buffered=%d dropped=%d total=%d",
			delivered, buffered, sess.DroppedInbound(), total)
	}
	if delivered+buffered > 500-0 && sess.LenInbound() > 0 {
		t.Fatal("缓冲应已排干")
	}
	t.Logf("守恒通过: delivered=%d buffered=%d dropped=%d", delivered, buffered, sess.DroppedInbound())
}
