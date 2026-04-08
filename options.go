package wecomaibot

import "time"

const (
	// DefaultWSURL is the production WebSocket endpoint.
	DefaultWSURL = "wss://openws.work.weixin.qq.com"

	// DefaultHeartbeatInterval is the recommended ping interval.
	DefaultHeartbeatInterval = 30 * time.Second

	// DefaultReconnectBaseDelay is the initial back-off delay.
	DefaultReconnectBaseDelay = 1 * time.Second

	// DefaultReconnectMaxDelay caps the exponential back-off.
	DefaultReconnectMaxDelay = 30 * time.Second

	// DefaultMaxReconnectAttempts limits retries; -1 means infinite.
	DefaultMaxReconnectAttempts = 10

	// DefaultRequestTimeout is the write deadline for a single frame.
	DefaultRequestTimeout = 10 * time.Second
)

// Options configures a Client.
type Options struct {
	// Required: the bot credentials.
	BotID  string
	Secret string

	// WebSocket endpoint. Defaults to DefaultWSURL.
	WSURL string

	// Heartbeat interval. Defaults to 30 s.
	HeartbeatInterval time.Duration

	// Reconnection back-off parameters.
	ReconnectBaseDelay   time.Duration
	ReconnectMaxDelay    time.Duration
	MaxReconnectAttempts int // -1 = infinite

	// Per-frame write timeout.
	RequestTimeout time.Duration

	// Logger instance; nil means the default stderr logger.
	Logger Logger
}

// withDefaults fills zero-valued fields with sensible defaults.
func (o Options) withDefaults() Options {
	if o.WSURL == "" {
		o.WSURL = DefaultWSURL
	}
	if o.HeartbeatInterval == 0 {
		o.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if o.ReconnectBaseDelay == 0 {
		o.ReconnectBaseDelay = DefaultReconnectBaseDelay
	}
	if o.ReconnectMaxDelay == 0 {
		o.ReconnectMaxDelay = DefaultReconnectMaxDelay
	}
	if o.MaxReconnectAttempts == 0 {
		o.MaxReconnectAttempts = DefaultMaxReconnectAttempts
	}
	if o.RequestTimeout == 0 {
		o.RequestTimeout = DefaultRequestTimeout
	}
	if o.Logger == nil {
		o.Logger = NewDefaultLogger()
	}
	return o
}
