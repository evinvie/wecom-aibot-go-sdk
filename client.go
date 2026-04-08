package wecomaibot

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client manages the WebSocket connection, authentication, heartbeat, and message dispatch.
type Client struct {
	opts      Options
	log       Logger
	emitter   *EventEmitter
	conn      *websocket.Conn
	connMu    sync.Mutex
	connected atomic.Bool
	stopCh    chan struct{}
	pending   sync.Map // req_id -> chan *Frame
}

// NewClient creates a new SDK client.
func NewClient(opts Options) *Client {
	opts = opts.withDefaults()
	return &Client{
		opts:    opts,
		log:     opts.Logger,
		emitter: NewEventEmitter(),
	}
}

// On registers an event handler. See event.go for event name constants.
func (c *Client) On(event string, h Handler) func() { return c.emitter.On(event, h) }

// IsConnected reports whether the WebSocket is currently connected.
func (c *Client) IsConnected() bool { return c.connected.Load() }

// GenerateReqID produces a unique request ID.
func GenerateReqID(prefix string) string {
	return prefix + "_" + uuid.New().String()[:8]
}

// ---------------------------------------------------------------------------
// Connection lifecycle
// ---------------------------------------------------------------------------

// Run connects and blocks until ctx is cancelled. It handles reconnection automatically.
func (c *Client) Run(ctx context.Context) error {
	c.stopCh = make(chan struct{})
	defer close(c.stopCh)
	return c.connectWithRetry(ctx)
}

func (c *Client) connectWithRetry(ctx context.Context) error {
	var attempt int
	for {
		err := c.connect(ctx)
		c.connected.Store(false)
		c.emitter.Emit(EventNameDisconnected, nil, nil)
		if err != nil {
			c.log.Error("connection lost: %v", err)
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
		attempt++
		if c.opts.MaxReconnectAttempts > 0 && attempt > c.opts.MaxReconnectAttempts {
			return fmt.Errorf("max reconnect attempts (%d) exceeded", c.opts.MaxReconnectAttempts)
		}

		delay := c.backoff(attempt)
		c.log.Info("reconnecting in %v (attempt %d)", delay, attempt)
		c.emitter.Emit(EventNameReconnecting, nil, attempt)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

func (c *Client) connect(ctx context.Context) error {
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, c.opts.WSURL, nil)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	c.connected.Store(true)
	c.emitter.Emit(EventNameConnected, nil, nil)
	c.log.Info("connected to %s", c.opts.WSURL)

	// Authenticate
	if err := c.authenticate(); err != nil {
		conn.Close()
		return err
	}

	// Start heartbeat
	heartCtx, heartCancel := context.WithCancel(ctx)
	defer heartCancel()
	go c.heartbeatLoop(heartCtx)

	// Read loop (blocks until error)
	return c.readLoop(ctx)
}

func (c *Client) authenticate() error {
	body := map[string]string{"bot_id": c.opts.BotID, "secret": c.opts.Secret}
	resp, err := c.Send(CmdSubscribe, body)
	if err != nil {
		return fmt.Errorf("subscribe send: %w", err)
	}
	if resp.ErrCode != 0 {
		return fmt.Errorf("subscribe rejected: %d %s", resp.ErrCode, resp.ErrMsg)
	}
	c.log.Info("authenticated (bot_id=%s)", c.opts.BotID)
	c.emitter.Emit(EventNameAuthenticated, nil, nil)
	return nil
}

// ---------------------------------------------------------------------------
// Heartbeat
// ---------------------------------------------------------------------------

func (c *Client) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(c.opts.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := c.Send(CmdPing, nil); err != nil {
				c.log.Warn("heartbeat failed: %v", err)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// Read loop & dispatch
// ---------------------------------------------------------------------------

func (c *Client) readLoop(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}
		var frame Frame
		if err := json.Unmarshal(msg, &frame); err != nil {
			c.log.Warn("unmarshal frame: %v", err)
			continue
		}

		// Check if this is a response to a pending request.
		if ch, ok := c.pending.LoadAndDelete(frame.Headers.ReqID); ok {
			ch.(chan *Frame) <- &frame
			continue
		}

		// Dispatch callbacks.
		go c.dispatch(&frame)
	}
}

func (c *Client) dispatch(frame *Frame) {
	defer func() {
		if r := recover(); r != nil {
			c.log.Error("handler panic: %v", r)
			c.emitter.Emit(EventNameError, frame, fmt.Errorf("handler panic: %v", r))
		}
	}()

	switch frame.Cmd {
	case CmdMsgCallback:
		var body MsgCallbackBody
		if err := json.Unmarshal(frame.Body, &body); err != nil {
			c.log.Warn("unmarshal msg callback: %v", err)
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
			c.log.Warn("unmarshal event callback: %v", err)
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
		}
	}
}

// ---------------------------------------------------------------------------
// Send / Reply helpers
// ---------------------------------------------------------------------------

// Send writes a frame and waits for the server response matching the req_id.
func (c *Client) Send(cmd string, body any) (*Frame, error) {
	reqID := GenerateReqID(cmd)

	var raw json.RawMessage
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
	}

	frame := Frame{
		Cmd:     cmd,
		Headers: Headers{ReqID: reqID},
		Body:    raw,
	}
	data, err := json.Marshal(frame)
	if err != nil {
		return nil, fmt.Errorf("marshal frame: %w", err)
	}

	ch := make(chan *Frame, 1)
	c.pending.Store(reqID, ch)

	c.connMu.Lock()
	writeErr := c.conn.WriteMessage(websocket.TextMessage, data)
	c.connMu.Unlock()
	if writeErr != nil {
		c.pending.Delete(reqID)
		return nil, fmt.Errorf("write: %w", writeErr)
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(c.opts.RequestTimeout):
		c.pending.Delete(reqID)
		return nil, fmt.Errorf("timeout waiting for response to %s", reqID)
	}
}

// sendReply is a fire-and-forget helper that uses the callback's req_id.
func (c *Client) sendReply(cmd, reqID string, body any) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	frame := Frame{Cmd: cmd, Headers: Headers{ReqID: reqID}, Body: raw}
	data, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("marshal frame: %w", err)
	}
	c.connMu.Lock()
	defer c.connMu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// Reply sends a generic reply to a callback frame.
func (c *Client) Reply(callbackFrame *Frame, body *ReplyBody) error {
	return c.sendReply(CmdRespondMsg, callbackFrame.Headers.ReqID, body)
}

// ReplyText is a convenience method for plain text replies.
func (c *Client) ReplyText(callbackFrame *Frame, content string) error {
	return c.Reply(callbackFrame, &ReplyBody{
		MsgType: MsgTypeText,
		Text:    &TextContent{Content: content},
	})
}

// ReplyMarkdown sends a Markdown-formatted reply.
func (c *Client) ReplyMarkdown(callbackFrame *Frame, content string) error {
	return c.Reply(callbackFrame, &ReplyBody{
		MsgType: MsgTypeMarkdown,
		Markdown: &MarkdownContent{Content: content},
	})
}

// ReplyStream sends or updates a streaming message. Set finish=true to end.
func (c *Client) ReplyStream(callbackFrame *Frame, streamID, content string, finish bool) error {
	return c.Reply(callbackFrame, &ReplyBody{
		MsgType: MsgTypeStream,
		Stream:  &StreamContent{ID: streamID, Finish: finish, Content: content},
	})
}

// ReplyTemplateCard sends a template card reply.
func (c *Client) ReplyTemplateCard(callbackFrame *Frame, card *TemplateCard) error {
	return c.Reply(callbackFrame, &ReplyBody{
		MsgType:      MsgTypeCard,
		TemplateCard: card,
	})
}

// ReplyWelcome sends a welcome message in response to an enter_chat event (within 5s).
func (c *Client) ReplyWelcome(callbackFrame *Frame, body *ReplyBody) error {
	return c.sendReply(CmdRespondWelcomeMsg, callbackFrame.Headers.ReqID, body)
}

// UpdateTemplateCard updates an existing card in response to a card click event (within 5s).
func (c *Client) UpdateTemplateCard(callbackFrame *Frame, card *TemplateCard) error {
	return c.sendReply(CmdRespondUpdateMsg, callbackFrame.Headers.ReqID, &UpdateCardBody{
		ResponseType: "update_template_card",
		TemplateCard: card,
	})
}

// SendMessage proactively pushes a message to a conversation.
func (c *Client) SendMessage(body *SendMsgBody) error {
	_, err := c.Send(CmdSendMsg, body)
	return err
}

// SendMarkdown is a convenience for pushing a Markdown message.
func (c *Client) SendMarkdown(chatID string, chatType ChatTypeInt, content string) error {
	return c.SendMessage(&SendMsgBody{
		ChatID: chatID, ChatType: chatType,
		MsgType:  MsgTypeMarkdown,
		Markdown: &MarkdownContent{Content: content},
	})
}

// Disconnect gracefully closes the connection.
func (c *Client) Disconnect() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// backoff calculates exponential back-off with a cap.
func (c *Client) backoff(attempt int) time.Duration {
	d := c.opts.ReconnectBaseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
	if d > c.opts.ReconnectMaxDelay {
		d = c.opts.ReconnectMaxDelay
	}
	return d
}
