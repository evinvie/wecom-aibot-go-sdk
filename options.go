package wecomaibot

import "time"

const (
	// DefaultWSURL 生产环境 WebSocket 端点地址。
	DefaultWSURL = "wss://openws.work.weixin.qq.com"

	// DefaultHeartbeatInterval 推荐的心跳间隔。
	DefaultHeartbeatInterval = 30 * time.Second

	// DefaultReconnectBaseDelay 指数退避的初始延迟。
	DefaultReconnectBaseDelay = 1 * time.Second

	// DefaultReconnectMaxDelay 指数退避的延迟上限。
	DefaultReconnectMaxDelay = 30 * time.Second

	// DefaultMaxReconnectAttempts 最大重连次数；-1 表示无限重连。
	DefaultMaxReconnectAttempts = 10

	// DefaultRequestTimeout 单帧写入超时时间。
	DefaultRequestTimeout = 10 * time.Second
)

// Options 用于配置 Client。
type Options struct {
	// 必填：机器人凭证。
	BotID  string
	Secret string

	// WebSocket 端点地址，默认为 DefaultWSURL。
	WSURL string

	// 心跳间隔，默认 30 秒。
	HeartbeatInterval time.Duration

	// 重连退避参数。
	ReconnectBaseDelay   time.Duration
	ReconnectMaxDelay    time.Duration
	MaxReconnectAttempts int // -1 = 无限重连

	// 单帧写入超时。
	RequestTimeout time.Duration

	// 日志实例；nil 使用默认 stderr 日志。
	Logger Logger
}

// withDefaults 为零值字段填充合理的默认值。
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
