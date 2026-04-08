package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	wecom "github.com/evinvie/wecom-aibot-go-sdk"
)

func main() {
	botID := os.Getenv("WECHAT_BOT_ID")
	secret := os.Getenv("WECHAT_BOT_SECRET")
	if botID == "" || secret == "" {
		log.Fatal("请设置环境变量 WECHAT_BOT_ID 和 WECHAT_BOT_SECRET")
	}

	client := wecom.NewClient(wecom.Options{
		BotID:  botID,
		Secret: secret,
	})

	// 监听认证成功事件
	client.On(wecom.EventNameAuthenticated, func(_ *wecom.Frame, _ any) {
		fmt.Println("==> 认证成功！")
	})

	// 处理文本消息 —— 流式回复示例
	client.On(wecom.EventNameMessageText, func(frame *wecom.Frame, payload any) {
		msg := payload.(*wecom.MsgCallbackBody)
		content := msg.Text.Content
		streamID := wecom.GenerateReqID("stream")

		_ = client.ReplyStream(frame, streamID, "正在思考中...", false)
		_ = client.ReplyStream(frame, streamID, fmt.Sprintf("你说的是: %q", content), true)
	})

	// 用户进入会话时发送欢迎语
	client.On(wecom.EventNameEnterChat, func(frame *wecom.Frame, _ any) {
		_ = client.ReplyWelcome(frame, &wecom.ReplyBody{
			MsgType: wecom.MsgTypeText,
			Text:    &wecom.TextContent{Content: "你好！有什么可以帮你的吗？"},
		})
	})

	// 优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("\n正在关闭...")
		cancel()
	}()

	if err := client.Run(ctx); err != nil {
		log.Printf("客户端已停止: %v", err)
	}
}
