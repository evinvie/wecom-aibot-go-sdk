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
		log.Fatal("WECHAT_BOT_ID and WECHAT_BOT_SECRET must be set")
	}

	client := wecom.NewClient(wecom.Options{
		BotID:  botID,
		Secret: secret,
	})

	// Lifecycle events
	client.On(wecom.EventNameAuthenticated, func(_ *wecom.Frame, _ any) {
		fmt.Println("==> Authenticated successfully!")
	})

	// Text message handler — echo with streaming
	client.On(wecom.EventNameMessageText, func(frame *wecom.Frame, payload any) {
		msg := payload.(*wecom.MsgCallbackBody)
		content := msg.Text.Content
		streamID := wecom.GenerateReqID("stream")

		_ = client.ReplyStream(frame, streamID, "Thinking...", false)
		_ = client.ReplyStream(frame, streamID, fmt.Sprintf("You said: %q", content), true)
	})

	// Welcome message on enter_chat
	client.On(wecom.EventNameEnterChat, func(frame *wecom.Frame, _ any) {
		_ = client.ReplyWelcome(frame, &wecom.ReplyBody{
			MsgType: wecom.MsgTypeText,
			Text:    &wecom.TextContent{Content: "Welcome! How can I help you?"},
		})
	})

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("\nShutting down...")
		cancel()
	}()

	if err := client.Run(ctx); err != nil {
		log.Printf("client stopped: %v", err)
	}
}
