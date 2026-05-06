# wecom-aibot-go-sdk

企业微信智能机器人 Go SDK —— 基于 WebSocket 长连接。

[![Go Reference](https://pkg.go.dev/badge/github.com/evinvie/wecom-aibot-go-sdk.svg)](https://pkg.go.dev/github.com/evinvie/wecom-aibot-go-sdk)

## 功能特性

| 特性 | 说明 |
|------|------|
| **WebSocket 长连接** | 自动连接 `wss://openws.work.weixin.qq.com`，无需公网 IP |
| **自动认证 & 心跳** | 内置 `aibot_subscribe` + 30s 心跳保活 |
| **指数退避重连** | 1s → 2s → 4s → ... → 30s，支持自定义最大重试次数 |
| **事件驱动** | 通过 `client.On(event, handler)` 监听所有消息和事件 |
| **流式回复** | 支持类 ChatGPT 逐字输出，Markdown 格式 |
| **模板卡片** | 发送和更新交互式卡片消息 |
| **主动推送** | 向指定会话推送 Markdown / 卡片 / 文件等 |
| **文件处理** | 分片上传（≤512KB/chunk）+ AES-256-CBC 解密下载 |
| **并发安全** | 所有公开方法均可在多 goroutine 中安全调用 |

## 安装

```bash
go get github.com/evinvie/wecom-aibot-go-sdk
```

> 要求 Go 1.21+

## 快速开始

```go
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
    client := wecom.NewClient(wecom.Options{
        BotID:  os.Getenv("WECHAT_BOT_ID"),
        Secret: os.Getenv("WECHAT_BOT_SECRET"),
    })

    // 认证成功
    client.On(wecom.EventNameAuthenticated, func(_ *wecom.Frame, _ any) {
        fmt.Println("🔐 认证成功!")
    })

    // 处理文本消息 —— 流式回复
    client.On(wecom.EventNameMessageText, func(frame *wecom.Frame, payload any) {
        msg := payload.(*wecom.MsgCallbackBody)
        streamID := wecom.GenerateReqID("stream")

        _ = client.ReplyStream(frame, streamID, "正在思考中...", false)
        _ = client.ReplyStream(frame, streamID,
            fmt.Sprintf("你好！你说的是: %q", msg.Text.Content), true)
    })

    // 进入会话时发送欢迎语（需 5s 内回复）
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
        cancel()
    }()

    if err := client.Run(ctx); err != nil {
        log.Printf("client stopped: %v", err)
    }
}
```

## 配置项

```go
wecom.Options{
    BotID:                "REQUIRED",           // 机器人 ID
    Secret:               "REQUIRED",           // 长连接密钥
    WSURL:                "wss://...",           // 默认: wss://openws.work.weixin.qq.com
    HeartbeatInterval:    30 * time.Second,      // 心跳间隔
    ReconnectBaseDelay:   1 * time.Second,       // 重连基础延迟
    ReconnectMaxDelay:    30 * time.Second,      // 重连最大延迟
    MaxReconnectAttempts: 10,                    // 最大重连次数 (-1 = 无限)
    RequestTimeout:       10 * time.Second,      // 帧级写超时
    Logger:               wecom.NewDefaultLogger(), // 自定义日志
}
```

## API 速查

### 事件监听

```go
// 连接生命周期
client.On("connected", handler)
client.On("authenticated", handler)
client.On("disconnected", handler)
client.On("reconnecting", handler)   // payload: int (attempt)
client.On("error", handler)          // payload: error

// 消息 (payload: *MsgCallbackBody)
client.On("message", handler)        // 全部消息
client.On("message.text", handler)
client.On("message.image", handler)
client.On("message.mixed", handler)
client.On("message.voice", handler)
client.On("message.file", handler)
client.On("message.video", handler)

// 事件 (payload: *EventCallbackBody)
client.On("event", handler)                    // 全部事件
client.On("event.enter_chat", handler)
client.On("event.template_card_event", handler)
client.On("event.feedback_event", handler)
```

### 回复消息

```go
// 纯文本
client.ReplyText(frame, "Hello!")

// Markdown
client.ReplyMarkdown(frame, "**加粗** 和 `代码`")

// 流式回复（基础方式）
streamID := wecom.GenerateReqID("stream")
client.ReplyStream(frame, streamID, "思考中...", false)
client.ReplyStream(frame, streamID, "最终答案", true)

// 流式回复（推荐：StreamSession 自动防 10 分钟超时）
stream := client.NewStream(frame)
stream.Update("正在处理...")        // 发送中间状态
stream.Update("继续分析...")        // 更新内容
if stream.IsExpired() {            // 可主动检查是否超时
    stream.Finish("处理超时，请重试")
}
stream.Finish("最终结果")           // 结束流式消息
// stream.Remaining() 可查看剩余时间

// 模板卡片
client.ReplyTemplateCard(frame, &wecom.TemplateCard{
    CardType:  "button_interaction",
    MainTitle: &wecom.CardTitle{Title: "告警通知", Desc: "CPU 超过 90%"},
    ButtonList: []wecom.CardButton{
        {Text: "确认", Style: 1, Key: "confirm"},
    },
    TaskID: "task_001",
})

// 欢迎语 (enter_chat 事件后 5s 内)
client.ReplyWelcome(frame, &wecom.ReplyBody{...})

// 更新卡片 (template_card_event 后 5s 内)
client.UpdateTemplateCard(frame, &wecom.TemplateCard{...})
```

### 主动推送

```go
// 推送 Markdown
client.SendMarkdown("CHAT_ID", wecom.ChatTypeIntSingle, "**提醒**：会议即将开始")

// 推送任意类型
client.SendMessage(&wecom.SendMsgBody{
    ChatID:   "CHAT_ID",
    ChatType: wecom.ChatTypeIntGroup,
    MsgType:  wecom.MsgTypeCard,
    TemplateCard: &wecom.TemplateCard{...},
})
```

### 文件操作

```go
// 下载并解密文件
data, filename, err := wecom.DownloadFile(imageURL, aesKey)

// 分片上传文件 → 获得 media_id
mediaID, err := client.UploadFile("file", "/path/to/report.pdf")

// 用 media_id 回复文件消息
client.Reply(frame, &wecom.ReplyBody{
    MsgType: wecom.MsgTypeFile,
    File:    &wecom.MediaContent{MediaID: mediaID},
})
```

## 项目结构

```
wecom-aibot-go-sdk/
├── types.go       # 所有数据结构 & 常量
├── options.go     # 配置项 & 默认值
├── logger.go      # Logger 接口 & 默认实现
├── event.go       # 事件总线 (EventEmitter)
├── client.go      # 核心客户端 (连接/认证/心跳/分发/回复)
├── media.go       # 文件上传/下载/AES 解密
├── example/
│   └── main.go    # 完整示例
├── go.mod
└── README.md
```

## 与 Python SDK 的对应关系

| Python SDK | Go SDK |
|------------|--------|
| `WSClient(options)` | `wecom.NewClient(opts)` |
| `@ws_client.on('message.text')` | `client.On("message.text", handler)` |
| `ws_client.reply_stream(frame, ...)` | `client.ReplyStream(frame, ...)` |
| `ws_client.reply_template_card(frame, ...)` | `client.ReplyTemplateCard(frame, ...)` |
| `ws_client.send_message(chatid, body)` | `client.SendMessage(&SendMsgBody{...})` |
| `ws_client.download_file(url, aes_key)` | `wecom.DownloadFile(url, aesKey)` |
| `ws_client.run()` | `client.Run(ctx)` |
| `generate_req_id('stream')` | `wecom.GenerateReqID("stream")` |

## 限制说明

- **连接限制**：每个机器人同时仅支持 1 个长连接，新连接会踢掉旧连接
- **频率限制**：30 条/分钟，1000 条/小时（单个会话）
- **流式消息**：首次发送后 10 分钟内必须结束（`finish=true`）
- **欢迎语 / 更新卡片**：收到事件后 5 秒内必须回复
- **上传文件**：单分片 ≤ 512KB，总分片 ≤ 100，文件有效期 3 天

## 自定义日志

实现 `Logger` 接口即可接入你的日志框架：

```go
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
}

// 使用 zap / slog / logrus 适配
client := wecom.NewClient(wecom.Options{
    Logger: myZapAdapter,
    // ...
})

// 禁用日志
client := wecom.NewClient(wecom.Options{
    Logger: wecom.NopLogger(),
})
```

## 协议

MIT License
