// wssmoke 对真实运行的网关做最小链路冒烟：
// 连接 WS → 发 N 帧音频 → 逐帧读回 → 校验内容一致 → PASS/FAIL。
// 前置：网关已启动，且 echoworker 已接管同一房间。
//
//	go run ./cmd/wssmoke -addr localhost:8080 -room smoke-1 -frames 100
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/coder/websocket"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "网关 HTTP 地址")
	roomID := flag.String("room", "smoke-1", "房间 ID（需有 worker 已接管）")
	frames := flag.Int("frames", 100, "发送帧数")
	intervalMs := flag.Int("interval-ms", 40, "发送间隔（毫秒）。40ms = 25fps 真实语音帧率；0 = 裸灌压测模式（网关会按水位丢帧，只断言不崩）")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	// 总超时 = 发送节奏耗时 + 余量
	ctx, cancel := context.WithTimeout(context.Background(),
		30*time.Second+time.Duration(*frames)*time.Duration(*intervalMs+200)*time.Millisecond)
	defer cancel()

	ws, _, err := websocket.Dial(ctx, fmt.Sprintf("ws://%s/ws?room_id=%s", *addr, *roomID), nil)
	if err != nil {
		log.Error("WS 连接失败", "err", err)
		os.Exit(1)
	}
	defer ws.Close(websocket.StatusNormalClosure, "")

	payload := make([]byte, 1280) // 40ms PCM 16kHz 16bit
	copy(payload, "smoke")

	sendStart := time.Now()
	for i := 0; i < *frames; i++ {
		if *intervalMs > 0 && i > 0 {
			time.Sleep(time.Duration(*intervalMs) * time.Millisecond)
		}
		if err := ws.Write(ctx, websocket.MessageBinary, payload); err != nil {
			log.Error("上行发送失败", "frame", i, "err", err)
			os.Exit(1)
		}
	}
	elapsed := time.Since(sendStart)

	for i := 0; i < *frames; i++ {
		msgType, data, err := ws.Read(ctx)
		if err != nil {
			log.Error("下行读取失败", "frame", i, "err", err)
			os.Exit(1)
		}
		if msgType != websocket.MessageBinary {
			log.Error("下行应为二进制帧", "frame", i, "type", msgType)
			os.Exit(1)
		}
		if string(data) != string(payload) {
			log.Error("下行帧内容不一致", "frame", i, "len", len(data))
			os.Exit(1)
		}
	}
	fmt.Printf("SMOKE PASS: %d 帧上行/下行逐帧一致 (room=%s, 发送耗时 %v)\n", *frames, *roomID, elapsed)
}
