package wecomaibot

import "sync"

// Event name constants.
const (
	EventNameConnected     = "connected"
	EventNameAuthenticated = "authenticated"
	EventNameDisconnected  = "disconnected"
	EventNameReconnecting  = "reconnecting"
	EventNameError         = "error"
	EventNameMessage       = "message"
	EventNameMessageText   = "message.text"
	EventNameMessageImage  = "message.image"
	EventNameMessageMixed  = "message.mixed"
	EventNameMessageVoice  = "message.voice"
	EventNameMessageFile   = "message.file"
	EventNameMessageVideo  = "message.video"
	EventNameEvent         = "event"
	EventNameEnterChat     = "event.enter_chat"
	EventNameTemplateCard  = "event.template_card_event"
	EventNameFeedbackEvent = "event.feedback_event"
)

// Handler is the callback for all events.
type Handler func(frame *Frame, payload any)

// EventEmitter is a thread-safe event bus.
type EventEmitter struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewEventEmitter() *EventEmitter {
	return &EventEmitter{handlers: make(map[string][]Handler)}
}

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

func (e *EventEmitter) Emit(event string, frame *Frame, payload any) {
	e.mu.RLock()
	hs := make([]Handler, len(e.handlers[event]))
	copy(hs, e.handlers[event])
	e.mu.RUnlock()
	for _, h := range hs {
		h(frame, payload)
	}
}
