package wecomaibot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// 哨兵错误，用于区分连接关闭的原因。
var (
	// ErrConnClosed 表示连接已关闭或不可用。
	ErrConnClosed = errors.New("wecom-aibot: 连接已关闭")

	// ErrDisconnectedByServer 表示被服务端踢掉（同一 BotID 的新连接建立）。
	ErrDisconnectedByServer = errors.New("wecom-aibot: 被服务端断开（新连接建立）")
)

// Client 管理 WebSocket 连接、认证、心跳和消息分发。
type Client struct {
	opts    Options
	log     Logger
	emitter *EventEmitter

	connMu    sync.Mutex       // 保护 conn 的读写（统一用互斥锁，避免 TOCTOU 竞态）
	conn      *websocket.Conn  // 当前连接，可能为 nil
	connected atomic.Bool      // 连接状态标记
	closeCh   chan struct{}     // 通知读循环退出（收到 disconnected_event 时关闭）
	stopCh    chan struct{}     // Run 退出时关闭
	pending   sync.Map         // req_id -> chan *Frame
}

// NewClient 创建一个新的 SDK 客户端。
func NewClient(opts Options) *Client {
	opts = opts.withDefaults()
	return &Client{
		opts:    opts,
		log:     opts.Logger,
		emitter: NewEventEmitter(),
	}
}

// On 注册事件处理函数。事件名称常量定义在 event.go 中。
func (c *Client) On(event string, h Handler) func() { return c.emitter.On(event, h) }

// IsConnected 返回当前 WebSocket 是否已连接。
func (c *Client) IsConnected() bool { return c.connected.Load() }

// GenerateReqID 生成唯一的请求 ID。
func GenerateReqID(prefix string) string {
	return prefix + "_" + uuid.New().String()[:8]
}

// ---------------------------------------------------------------------------
// 连接生命周期
// ---------------------------------------------------------------------------

// Run 建立连接并阻塞直到 ctx 被取消。自动处理断线重连。
func (c *Client) Run(ctx context.Context) error {
	c.stopCh = make(chan struct{})
	defer close(c.stopCh)
	return c.connectWithRetry(ctx)
}

// connectWithRetry 带重连的连接循环。
func (c *Client) connectWithRetry(ctx context.Context) error {
	var attempt int
	for {
		connectedAt := time.Now()
		err := c.connect(ctx)
		c.connected.Store(false)

		// 如果这次连接持续了超过 1 分钟，说明之前重连成功过，重置计数器
		if time.Since(connectedAt) > time.Minute {
			attempt = 0
		}

		// 只在 dispatch 未触发过 disconnected 事件时才 Emit，避免重复通知用户
		if !errors.Is(err, ErrDisconnectedByServer) {
			c.emitter.Emit(EventNameDisconnected, nil, nil)
		}
		if err != nil {
			c.log.Error("连接断开: %v", err)
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
		attempt++
		if c.opts.MaxReconnectAttempts > 0 && attempt > c.opts.MaxReconnectAttempts {
			return fmt.Errorf("超过最大重连次数 (%d)", c.opts.MaxReconnectAttempts)
		}

		delay := c.backoff(attempt)
		c.log.Info("将在 %v 后重连 (第 %d 次)", delay, attempt)
		c.emitter.Emit(EventNameReconnecting, nil, attempt)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

// connect 执行一次完整的连接流程：拨号 → 认证 → 心跳 → 读取循环。
func (c *Client) connect(ctx context.Context) error {
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, c.opts.WSURL, nil)
	if err != nil {
		return fmt.Errorf("拨号失败: %w", err)
	}

	// 初始化本轮连接状态
	c.closeCh = make(chan struct{})

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	c.connected.Store(true)
	c.emitter.Emit(EventNameConnected, nil, nil)
	c.log.Info("已连接到 %s", c.opts.WSURL)

	// 认证
	if err := c.authenticate(); err != nil {
		c.closeConn()
		return err
	}

	// 启动心跳
	heartCtx, heartCancel := context.WithCancel(ctx)
	defer heartCancel()
	go c.heartbeatLoop(heartCtx)

	// 读取循环（阻塞直到出错）
	return c.readLoop(ctx)
}

// authenticate 发送订阅请求完成身份认证。
func (c *Client) authenticate() error {
	body := map[string]string{"bot_id": c.opts.BotID, "secret": c.opts.Secret}
	resp, err := c.Send(CmdSubscribe, body)
	if err != nil {
		return fmt.Errorf("订阅请求发送失败: %w", err)
	}
	if resp.ErrCode != 0 {
		return fmt.Errorf("订阅被拒绝: %d %s", resp.ErrCode, resp.ErrMsg)
	}
	c.log.Info("认证成功 (bot_id=%s)", c.opts.BotID)
	c.emitter.Emit(EventNameAuthenticated, nil, nil)
	return nil
}

// ---------------------------------------------------------------------------
// 心跳保活
// ---------------------------------------------------------------------------

// heartbeatLoop 定期发送心跳帧。内部 goroutine，带 recover 兜底。
// 心跳使用 fire-and-forget 方式，不等待响应（服务端不一定回复 ping）。
func (c *Client) heartbeatLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			c.log.Error("心跳 goroutine panic: %v", r)
		}
	}()

	ticker := time.NewTicker(c.opts.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.sendPing(); err != nil {
				c.log.Warn("心跳发送失败: %v", err)
				if errors.Is(err, ErrConnClosed) {
					return
				}
			}
		}
	}
}

// sendPing 发送心跳帧（fire-and-forget，不等待响应）。
func (c *Client) sendPing() error {
	frame := Frame{
		Cmd:     CmdPing,
		Headers: Headers{ReqID: GenerateReqID("ping")},
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("序列化心跳帧失败: %w", err)
	}
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn == nil {
		return ErrConnClosed
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// ---------------------------------------------------------------------------
// 读取循环与消息分发
// ---------------------------------------------------------------------------

// readLoop 持续读取 WebSocket 消息并进行分发。
func (c *Client) readLoop(ctx context.Context) error {
	for {
		// 优先检查退出信号
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closeCh:
			return ErrDisconnectedByServer
		default:
		}

		// 加锁获取 conn 引用，避免 nil 解引用
		c.connMu.Lock()
		conn := c.conn
		c.connMu.Unlock()
		if conn == nil {
			return ErrConnClosed
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			select {
			case <-c.closeCh:
				return ErrDisconnectedByServer
			default:
			}
			return fmt.Errorf("读取消息失败: %w", err)
		}

		var frame Frame
		if err := json.Unmarshal(msg, &frame); err != nil {
			c.log.Warn("帧解析失败: %v", err)
			continue
		}

		// 检查是否是待处理请求的响应
		if ch, ok := c.pending.LoadAndDelete(frame.Headers.ReqID); ok {
			ch.(chan *Frame) <- &frame
			continue
		}

		// 异步分发回调（带 recover）
		go c.dispatch(&frame)
	}
}

// dispatch 根据命令类型分发帧到对应的事件处理函数。
func (c *Client) dispatch(frame *Frame) {
	defer func() {
		if r := recover(); r != nil {
			c.log.Error("处理函数发生 panic: %v", r)
			c.emitter.Emit(EventNameError, frame, fmt.Errorf("处理函数 panic: %v", r))
		}
	}()

	switch frame.Cmd {
	case CmdMsgCallback:
		var body MsgCallbackBody
		if err := json.Unmarshal(frame.Body, &body); err != nil {
			c.log.Warn("消息回调解析失败: %v", err)
			return
		}
		c.emitter.Emit(EventNameMessage, frame, &body)
		switch body.MsgType {
		case MsgTypeText:
			c.emitter.Emit(EventNameMessageText, frame, &body)
		case MsgTypeImage:
			c.emitter.Emit(EventNameMessageImage, frame, &body)
		case MsgTypeMixed:
			c.emitter.Emit(EventNameMessageMixed, frame, &body)
		case MsgTypeVoice:
			c.emitter.Emit(EventNameMessageVoice, frame, &body)
		case MsgTypeFile:
			c.emitter.Emit(EventNameMessageFile, frame, &body)
		case MsgTypeVideo:
			c.emitter.Emit(EventNameMessageVideo, frame, &body)
		}

	case CmdEventCallback:
		var body EventCallbackBody
		if err := json.Unmarshal(frame.Body, &body); err != nil {
			c.log.Warn("事件回调解析失败: %v", err)
			return
		}
		c.emitter.Emit(EventNameEvent, frame, &body)
		switch body.Event.EventType {
		case EventEnterChat:
			c.emitter.Emit(EventNameEnterChat, frame, &body)
		case EventTemplateCard:
			c.emitter.Emit(EventNameTemplateCard, frame, &body)
		case EventFeedback:
			c.emitter.Emit(EventNameFeedbackEvent, frame, &body)
		case EventDisconnected:
			// 被服务端踢下线：先触发事件，再安全关闭连接让读循环退出
			c.log.Warn("收到服务端断开通知 (disconnected_event)")
			c.emitter.Emit(EventNameDisconnected, frame, &body)
			c.closeConn()
		}
	}
}

// ---------------------------------------------------------------------------
// 发送与回复
// ---------------------------------------------------------------------------

// Send 发送一个帧并等待服务端响应（通过 req_id 匹配）。
func (c *Client) Send(cmd string, body any) (*Frame, error) {
	reqID := GenerateReqID(cmd)

	var raw json.RawMessage
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("序列化消息体失败: %w", err)
		}
	}

	frame := Frame{
		Cmd:     cmd,
		Headers: Headers{ReqID: reqID},
		Body:    raw,
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("序列化帧失败: %w", err)
	}

	ch := make(chan *Frame, 1)
	c.pending.Store(reqID, ch)

	// 加锁检查 + 写入，原子操作避免 TOCTOU 竞态
	c.connMu.Lock()
	if c.conn == nil {
		c.connMu.Unlock()
		c.pending.Delete(reqID)
		return nil, ErrConnClosed
	}
	writeErr := c.conn.WriteMessage(websocket.TextMessage, data)
	c.connMu.Unlock()

	if writeErr != nil {
		c.pending.Delete(reqID)
		return nil, fmt.Errorf("写入失败: %w", writeErr)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(c.opts.RequestTimeout):
		c.pending.Delete(reqID)
		return nil, fmt.Errorf("等待 %s 的响应超时", reqID)
	}
}

// sendReply 使用回调帧的 req_id 发送回复（不等待响应）。
func (c *Client) sendReply(cmd, reqID string, body any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("序列化消息体失败: %w", err)
	}
	frame := Frame{Cmd: cmd, Headers: Headers{ReqID: reqID}, Body: raw}
	data, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("序列化帧失败: %w", err)
	}

	// 加锁检查 + 写入，原子操作
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn == nil {
		return ErrConnClosed
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// Reply 向回调帧发送通用回复。
func (c *Client) Reply(callbackFrame *Frame, body *ReplyBody) error {
	return c.sendReply(CmdRespondMsg, callbackFrame.Headers.ReqID, body)
}

// ReplyText 回复纯文本消息的便捷方法。
func (c *Client) ReplyText(callbackFrame *Frame, content string) error {
	return c.Reply(callbackFrame, &ReplyBody{
		MsgType: MsgTypeText,
		Text:    &TextContent{Content: content},
	})
}

// ReplyMarkdown 回复 Markdown 格式消息。
func (c *Client) ReplyMarkdown(callbackFrame *Frame, content string) error {
	return c.Reply(callbackFrame, &ReplyBody{
		MsgType:  MsgTypeMarkdown,
		Markdown: &MarkdownContent{Content: content},
	})
}

// ReplyStream 发送或更新流式消息。设置 finish=true 结束流式输出。
func (c *Client) ReplyStream(callbackFrame *Frame, streamID, content string, finish bool) error {
	return c.Reply(callbackFrame, &ReplyBody{
		MsgType: MsgTypeStream,
		Stream:  &StreamContent{ID: streamID, Finish: finish, Content: content},
	})
}

// ReplyTemplateCard 回复模板卡片消息。
func (c *Client) ReplyTemplateCard(callbackFrame *Frame, card *TemplateCard) error {
	return c.Reply(callbackFrame, &ReplyBody{
		MsgType:      MsgTypeCard,
		TemplateCard: card,
	})
}

// ReplyWelcome 发送欢迎语（需在收到 enter_chat 事件后 5 秒内调用）。
func (c *Client) ReplyWelcome(callbackFrame *Frame, body *ReplyBody) error {
	return c.sendReply(CmdRespondWelcomeMsg, callbackFrame.Headers.ReqID, body)
}

// UpdateTemplateCard 更新已有的模板卡片（需在收到卡片点击事件后 5 秒内调用）。
func (c *Client) UpdateTemplateCard(callbackFrame *Frame, card *TemplateCard) error {
	return c.sendReply(CmdRespondUpdateMsg, callbackFrame.Headers.ReqID, &UpdateCardBody{
		ResponseType: "update_template_card",
		TemplateCard: card,
	})
}

// SendMessage 主动向会话推送消息。
func (c *Client) SendMessage(body *SendMsgBody) error {
	_, err := c.Send(CmdSendMsg, body)
	return err
}

// SendMarkdown 主动推送 Markdown 消息的便捷方法。
func (c *Client) SendMarkdown(chatID string, chatType ChatTypeInt, content string) error {
	return c.SendMessage(&SendMsgBody{
		ChatID: chatID, ChatType: chatType,
		MsgType:  MsgTypeMarkdown,
		Markdown: &MarkdownContent{Content: content},
	})
}

// ---------------------------------------------------------------------------
// 连接关闭
// ---------------------------------------------------------------------------

// closeConn 安全关闭当前连接并通知读循环退出。
// 可被多个 goroutine 安全调用（幂等）。
func (c *Client) closeConn() {
	c.connected.Store(false)

	// 通知读循环退出
	select {
	case <-c.closeCh:
		// 已经关闭过了
	default:
		close(c.closeCh)
	}

	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	// 清理所有待处理的请求，避免 goroutine 泄漏
	c.pending.Range(func(key, value any) bool {
		c.pending.Delete(key)
		ch := value.(chan *Frame)
		select {
		case ch <- &Frame{ErrCode: -1, ErrMsg: "连接已关闭"}:
		default:
		}
		return true
	})
}

// Disconnect 优雅地关闭 WebSocket 连接。
func (c *Client) Disconnect() error {
	c.closeConn()
	return nil
}

// backoff 计算指数退避延迟（带上限）。
func (c *Client) backoff(attempt int) time.Duration {
	d := c.opts.ReconnectBaseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
	if d > c.opts.ReconnectMaxDelay {
		d = c.opts.ReconnectMaxDelay
	}
	return d
}
