package wecomaibot

import (
	"fmt"
	"os"
	"strings"
	"time"
)

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
	// 必填：机器人凭证（二选一设置）。
	//
	// 方式一：直接传值（不推荐提交到代码仓库）。
	// 方式二：使用 SecretFile / BotIDFile 从文件读取（推荐）。
	BotID  string
	Secret string

	// 可选：从文件读取凭证（优先级高于直接赋值）。
	// 文件内容为纯文本，读取后会去除首尾空白。
	// 推荐：将文件权限设为 600 (仅 owner 可读写)。
	SecretFile string // 密钥文件路径
	BotIDFile  string // BotID 文件路径

	// WebSocket 端点地址，默认为 DefaultWSURL。
	WSURL string

	// 心跳间隔，默认 30 秒。
	HeartbeatInterval time.Duration

	// 重连退避参数。
	ReconnectBaseDelay   time.Duration
	ReconnectMaxDelay    time.Duration
	MaxReconnectAttempts int // >0: 限制重连次数; 0: 使用默认值(10); -1: 无限重连

	// 单帧写入超时。
	RequestTimeout time.Duration

	// 日志实例；nil 使用默认 stderr 日志。
	Logger Logger
}

// withDefaults 为零值字段填充合理的默认值，并从文件加载凭证。
func (o Options) withDefaults() (Options, error) {
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

	// 从文件加载凭证（优先级高于直接赋值）
	if o.BotIDFile != "" {
		data, err := os.ReadFile(o.BotIDFile)
		if err != nil {
			return o, fmt.Errorf("读取 BotID 文件失败 (%s): %w", o.BotIDFile, err)
		}
		o.BotID = strings.TrimSpace(string(data))
	}
	if o.SecretFile != "" {
		data, err := os.ReadFile(o.SecretFile)
		if err != nil {
			return o, fmt.Errorf("读取 Secret 文件失败 (%s): %w", o.SecretFile, err)
		}
		o.Secret = strings.TrimSpace(string(data))
	}

	return o, nil
}

// maskSecret 返回 secret 的脱敏形式，仅保留前4位和后4位。
// 用于日志输出，避免泄露完整密钥。
func maskSecret(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}
