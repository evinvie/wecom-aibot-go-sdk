package wecomaibot

import "sync"

// 事件名称常量。
const (
	EventNameConnected     = "connected"                 // 连接建立
	EventNameAuthenticated = "authenticated"              // 认证成功
	EventNameDisconnected  = "disconnected"               // 连接断开
	EventNameReconnecting  = "reconnecting"               // 正在重连
	EventNameError         = "error"                      // 发生错误
	EventNameMessage       = "message"                    // 所有消息
	EventNameMessageText   = "message.text"               // 文本消息
	EventNameMessageImage  = "message.image"              // 图片消息
	EventNameMessageMixed  = "message.mixed"              // 图文混排消息
	EventNameMessageVoice  = "message.voice"              // 语音消息
	EventNameMessageFile   = "message.file"               // 文件消息
	EventNameMessageVideo  = "message.video"              // 视频消息
	EventNameEvent         = "event"                      // 所有事件
	EventNameEnterChat     = "event.enter_chat"           // 用户进入会话
	EventNameTemplateCard  = "event.template_card_event"  // 模板卡片点击
	EventNameFeedbackEvent = "event.feedback_event"       // 用户反馈
)

// Handler 是所有事件的回调函数签名。
type Handler func(frame *Frame, payload any)

// handlerEntry 将 handler 与唯一 ID 绑定，用于安全注销。
type handlerEntry struct {
	id uint64
	fn Handler
}

// EventEmitter 是并发安全的事件总线。
type EventEmitter struct {
	mu       sync.RWMutex
	handlers map[string][]handlerEntry
	nextID   uint64
}

// NewEventEmitter 创建一个空的事件发射器。
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{handlers: make(map[string][]handlerEntry)}
}

// On 注册指定事件的处理函数，返回一个可取消注册的函数。
// 多次调用返回的取消函数是安全的（幂等）。
func (e *EventEmitter) On(event string, h Handler) func() {
	e.mu.Lock()
	e.nextID++
	id := e.nextID
	e.handlers[event] = append(e.handlers[event], handlerEntry{id: id, fn: h})
	e.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			e.mu.Lock()
			defer e.mu.Unlock()
			hs := e.handlers[event]
			for i, entry := range hs {
				if entry.id == id {
					e.handlers[event] = append(hs[:i], hs[i+1:]...)
					break
				}
			}
		})
	}
}

// Emit 同步触发指定事件的所有处理函数。
func (e *EventEmitter) Emit(event string, frame *Frame, payload any) {
	e.mu.RLock()
	hs := make([]handlerEntry, len(e.handlers[event]))
	copy(hs, e.handlers[event])
	e.mu.RUnlock()
	for _, entry := range hs {
		entry.fn(frame, payload)
	}
}
