package integration

// M1 端到端集成测试：WS 客户端 → 网关 → 回声 worker（gRPC）→ 网关 → WS 客户端。
// 网关进程内组装（httptest + bufconn），验证 SPEC「Minimal Verifiable Slice」。

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"media-gateway/gen/mediav1"
	"media-gateway/internal/config"
	"media-gateway/internal/registry"
	"media-gateway/internal/rpc"
	"media-gateway/internal/server"
)

const frameSize = 1280 // 40ms PCM 16kHz 16bit

// startStack 组装完整网关（WS over httptest + gRPC over bufconn），返回 WS URL 与 worker 拨号函数
func startStack(t *testing.T, sessionBuffer int) (string, func(context.Context) (*grpc.ClientConn, error)) {
	t.Helper()
	cfg := config.Default()
	cfg.SessionBuffer = sessionBuffer
	cfg.FirstFrameSec = 5 * time.Second
	cfg.IdleTimeout = 10 * time.Second

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reg := registry.NewLocal()

	// gRPC over bufconn
	grpcSrv := grpc.NewServer()
	mediav1.RegisterMediaRelayServer(grpcSrv, &rpc.RelayServer{Registry: reg, Logger: log})
	ln := bufconn.Listen(1 << 20)
	go func() { _ = grpcSrv.Serve(ln) }()
	t.Cleanup(grpcSrv.Stop)

	dialWorker := func(ctx context.Context) (*grpc.ClientConn, error) {
		return grpc.NewClient("passthrough:///bufnet",
			grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				return ln.DialContext(ctx)
			}),
			grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// WS over httptest
	mux := http.NewServeMux()
	mux.Handle("/ws", server.NewWSHandler(cfg, reg, log))
	httpSrv := httptest.NewServer(mux)
	t.Cleanup(httpSrv.Close)

	return "ws" + httpSrv.URL[4:] + "/ws", dialWorker
}

// runEchoWorker 模拟回声 worker：start 后把上行音频逐帧回放
func runEchoWorker(ctx context.Context, t *testing.T, dial func(context.Context) (*grpc.ClientConn, error), roomID string) {
	t.Helper()
	conn, err := dial(ctx)
	if err != nil {
		t.Fatalf("worker 拨号失败: %v", err)
	}
	stream, err := mediav1.NewMediaRelayClient(conn).SessionStream(ctx)
	if err != nil {
		t.Fatalf("worker 建流失败: %v", err)
	}
	if err := stream.Send(&mediav1.WorkerFrame{Payload: &mediav1.WorkerFrame_Control{
		Control: &mediav1.StreamControl{RoomId: roomID, Event: "start"},
	}}); err != nil {
		t.Fatalf("worker start 失败: %v", err)
	}
	go func() {
		for {
			gf, err := stream.Recv()
			if err != nil {
				return
			}
			if a := gf.GetAudio(); a != nil {
				_ = stream.Send(&mediav1.WorkerFrame{Payload: &mediav1.WorkerFrame_TtsAudio{
					TtsAudio: a,
				}})
			}
		}
	}()
}

// TestEchoLoop 全链路回声：50 帧上行 → worker → 50 帧下行，字节与 seq 双向一致
func TestEchoLoop(t *testing.T) {
	const roomID = "it-echo"
	const totalFrames = 50

	wsURL, dial := startStack(t, 100) // 缓冲调大，避免测试灌帧快于回声导致水位丢帧
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	runEchoWorker(ctx, t, dial, roomID)

	wsConn, _, err := websocket.Dial(ctx, wsURL+"?room_id="+roomID, nil)
	if err != nil {
		t.Fatalf("WS 连接失败: %v", err)
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "")

	// 上行：50 帧可区分字节（帧尾嵌序号）
	sentPayloads := make(map[int][]byte)
	for i := 0; i < totalFrames; i++ {
		payload := make([]byte, frameSize)
		copy(payload, fmt.Appendf(nil, "frame-%05d", i))
		sentPayloads[i] = payload
		if err := wsConn.Write(ctx, websocket.MessageBinary, payload); err != nil {
			t.Fatalf("上行发送失败: %v", err)
		}
	}

	// 下行：逐帧校验内容一致
	for i := 0; i < totalFrames; i++ {
		msgType, data, err := wsConn.Read(ctx)
		if err != nil {
			t.Fatalf("下行读取失败（第 %d 帧）: %v", i, err)
		}
		if msgType != websocket.MessageBinary {
			t.Fatalf("第 %d 帧应为二进制，got %v: %s", i, msgType, data)
		}
		if !bytes.Equal(data, sentPayloads[i]) {
			t.Fatalf("第 %d 帧回声内容不一致", i)
		}
	}
}

// TestDuplicateRoomRejected 同房间第二个连接被拒绝
func TestDuplicateRoomRejected(t *testing.T) {
	const roomID = "it-dup"
	wsURL, dial := startStack(t, 50)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	runEchoWorker(ctx, t, dial, roomID)

	c1, _, err := websocket.Dial(ctx, wsURL+"?room_id="+roomID, nil)
	if err != nil {
		t.Fatalf("第一个连接应成功: %v", err)
	}
	defer c1.Close(websocket.StatusNormalClosure, "")

	c2, _, err := websocket.Dial(ctx, wsURL+"?room_id="+roomID, nil)
	if err != nil {
		t.Fatalf("第二个连接的 HTTP 升级可能成功，拒绝发生在 close 帧: %v", err)
	}
	defer c2.Close(websocket.StatusNormalClosure, "")

	// 拒绝以 close 帧形式到达
	_, _, err = c2.Read(ctx)
	if err == nil {
		t.Fatal("第二个连接应收到 close 帧")
	}
}

// TestStreamRequiresStartControl worker 首帧不是控制帧应报错
func TestStreamRequiresStartControl(t *testing.T) {
	wsURL, dial := startStack(t, 50)
	_ = wsURL
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := dial(ctx)
	if err != nil {
		t.Fatalf("拨号失败: %v", err)
	}
	defer conn.Close()

	stream, err := mediav1.NewMediaRelayClient(conn).SessionStream(ctx)
	if err != nil {
		t.Fatalf("建流失败: %v", err)
	}
	// 首帧直接发音频
	_ = stream.Send(&mediav1.WorkerFrame{Payload: &mediav1.WorkerFrame_TtsAudio{
		TtsAudio: &mediav1.AudioChunk{RoomId: "r", Data: []byte("x")},
	}})
	_, err = stream.Recv()
	if err == nil {
		t.Fatal("无 start 控制帧应返回错误")
	}
}
