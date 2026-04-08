package wecomaibot

import "sync"

// 事件名称常量。
const (
	EventNameConnected     = "connected"              // 连接建立
	EventNameAuthenticated = "authenticated"           // 认证成功
	EventNameDisconnected  = "disconnected"            // 连接断开
	EventNameReconnecting  = "reconnecting"            // 正在重连
	EventNameError         = "error"                   // 发生错误
	EventNameMessage       = "message"                 // 所有消息
	EventNameMessageText   = "message.text"            // 文本消息
	EventNameMessageImage  = "message.image"           // 图片消息
	EventNameMessageMixed  = "message.mixed"           // 图文混排消息
	EventNameMessageVoice  = "message.voice"           // 语音消息
	EventNameMessageFile   = "message.file"            // 文件消息
	EventNameMessageVideo  = "message.video"           // 视频消息
	EventNameEvent         = "event"                   // 所有事件
	EventNameEnterChat     = "event.enter_chat"        // 用户进入会话
	EventNameTemplateCard  = "event.template_card_event" // 模板卡片点击
	EventNameFeedbackEvent = "event.feedback_event"    // 用户反馈
)

// Handler 是所有事件的回调函数签名。
type Handler func(frame *Frame, payload any)

// EventEmitter 是并发安全的事件总线。
type EventEmitter struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewEventEmitter 创建一个空的事件发射器。
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{handlers: make(map[string][]Handler)}
}

// On 注册指定事件的处理函数，返回一个可取消注册的函数。
func (e *EventEmitter) On(event string, h Handler) func() {
	e.mu.Lock()
	e.handlers[event] = append(e.handlers[event], h)
	idx := len(e.handlers[event]) - 1
	e.mu.Unlock()
	return func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		hs := e.handlers[event]
		if idx < len(hs) {
			e.handlers[event] = append(hs[:idx], hs[idx+1:]...)
		}
	}
}

// Emit 同步触发指定事件的所有处理函数。
func (e *EventEmitter) Emit(event string, frame *Frame, payload any) {
	e.mu.RLock()
	hs := make([]Handler, len(e.handlers[event]))
	copy(hs, e.handlers[event])
	e.mu.RUnlock()
	for _, h := range hs {
		h(frame, payload)
	}
}
