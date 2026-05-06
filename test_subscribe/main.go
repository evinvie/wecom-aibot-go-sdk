package main

import (
	"context"
	"fmt"
	"os"
	"time"

	wecom "github.com/evinvie/wecom-aibot-go-sdk"
)

func main() {
	client := wecom.NewClient(wecom.Options{
		BotID:  "aibuDpwikMdj_29wT2CeCpi6f2IHUOALxsp",
		Secret: "A00labArIGW02gGtErzOPcCNSeU1Fpb3U4CPkP2F48b",
		MaxReconnectAttempts: 1, // 只试一次，测完就退
		RequestTimeout:       10 * time.Second,
	})

	client.On(wecom.EventNameConnected, func(_ *wecom.Frame, _ any) {
		fmt.Println("✅ WebSocket 连接建立成功")
	})

	client.On(wecom.EventNameAuthenticated, func(_ *wecom.Frame, _ any) {
		fmt.Println("✅ 订阅认证成功！aibot_subscribe 返回 errcode=0")
		fmt.Println("测试通过，5秒后退出...")
		go func() {
			time.Sleep(5 * time.Second)
			os.Exit(0)
		}()
	})

	client.On(wecom.EventNameError, func(_ *wecom.Frame, payload any) {
		fmt.Printf("❌ 错误: %v\n", payload)
	})

	client.On(wecom.EventNameDisconnected, func(_ *wecom.Frame, _ any) {
		fmt.Println("⚠️ 连接断开")
	})

	client.On(wecom.EventNameMessageText, func(frame *wecom.Frame, payload any) {
		msg := payload.(*wecom.MsgCallbackBody)
		fmt.Printf("📩 收到文本消息: %s (来自: %s)\n", msg.Text.Content, msg.From.UserID)
	})

	fmt.Println("正在连接 wss://openws.work.weixin.qq.com ...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Run(ctx); err != nil {
		fmt.Printf("客户端退出: %v\n", err)
	}
}
