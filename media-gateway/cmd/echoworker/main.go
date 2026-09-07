// echoworker 是 M1 验证用 mock worker：接入网关后把浏览器上行音频原样回放（TTS 下行）。
// 用于在接真实 ASR/Agent/TTS 之前验证媒体链路与压测（SPEC「Minimal Verifiable Slice」）。
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"media-gateway/gen/mediav1"
)

func main() {
	gatewayAddr := flag.String("gateway", "localhost:9090", "网关 gRPC 地址")
	roomID := flag.String("room", "echo-1", "要接管的房间 ID")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	conn, err := grpc.NewClient(*gatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("连接网关失败", "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	// 拨号重试：网关滚动发布/启动时序下 worker 必须能自动重挂
	var stream mediav1.MediaRelay_SessionStreamClient
	for attempt := 1; ; attempt++ {
		stream, err = mediav1.NewMediaRelayClient(conn).SessionStream(ctx)
		if err == nil {
			break
		}
		if attempt >= 20 {
			log.Error("建立会话流失败（重试耗尽）", "err", err)
			os.Exit(1)
		}
		time.Sleep(500 * time.Millisecond)
	}

	if err := stream.Send(&mediav1.WorkerFrame{Payload: &mediav1.WorkerFrame_Control{
		Control: &mediav1.StreamControl{RoomId: *roomID, Event: "start"},
	}}); err != nil {
		log.Error("发送 start 失败", "err", err)
		os.Exit(1)
	}
	log.Info("回声 worker 已接入", "room_id", *roomID, "gateway", *gatewayAddr)

	echoed := 0
	for {
		gf, err := stream.Recv()
		if err != nil {
			if ctx.Err() == nil {
				log.Info("网关流结束", "err", err)
			}
			return
		}
		switch p := gf.Payload.(type) {
		case *mediav1.GatewayFrame_Audio:
			a := p.Audio
			// 回声：原样作为 TTS 音频推回浏览器
			err := stream.Send(&mediav1.WorkerFrame{Payload: &mediav1.WorkerFrame_TtsAudio{
				TtsAudio: &mediav1.AudioChunk{
					RoomId: a.RoomId, Seq: a.Seq, TimestampMs: a.TimestampMs,
					Codec: a.Codec, Data: a.Data, Final: a.Final,
				},
			}})
			if err != nil {
				log.Warn("回声帧发送失败", "seq", a.Seq, "err", err)
				return
			}
			echoed++
			if echoed%100 == 0 {
				log.Info("回声中", "echoed", echoed, "last_seq", a.Seq)
			}
		case *mediav1.GatewayFrame_Ended:
			log.Info("会话结束", "reason", p.Ended.Reason, "echoed", echoed)
			return
		default:
			log.Warn("未知帧类型，忽略")
		}
	}
}
