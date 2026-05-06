package wecomaibot

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// 流式消息的服务端限制常量。
const (
	// StreamMaxDuration 是服务端允许的流式消息最大持续时间（从首次发送算起）。
	StreamMaxDuration = 10 * time.Minute

	// StreamWarnThreshold 是接近超时时发出警告的阈值（距超时 30 秒时警告）。
	StreamWarnThreshold = 30 * time.Second
)

// ErrStreamExpired 表示流式消息已超过 10 分钟限制。
var ErrStreamExpired = errors.New("wecom-aibot: 流式消息已超过 10 分钟限制，服务端将拒绝更新")

// StreamSession 管理一条流式消息的生命周期，自动追踪时间防止超过 10 分钟限制。
//
// 使用方式:
//
//	stream := client.NewStream(callbackFrame)
//	stream.Update("正在处理...")    // 发送中间状态
//	stream.Update("继续处理...")    // 更新内容
//	stream.Finish("最终结果")       // 结束流式消息
type StreamSession struct {
	client    *Client
	frame     *Frame
	streamID  string
	startTime time.Time
	finished  bool
	mu        sync.Mutex
	log       Logger
}

// NewStream 创建一个新的流式消息会话。
// 内部自动生成 stream_id 并记录起始时间。
func (c *Client) NewStream(callbackFrame *Frame) *StreamSession {
	return &StreamSession{
		client:   c,
		frame:    callbackFrame,
		streamID: GenerateReqID("stream"),
		log:      c.log,
	}
}

// NewStreamWithID 创建一个使用指定 streamID 的流式消息会话。
func (c *Client) NewStreamWithID(callbackFrame *Frame, streamID string) *StreamSession {
	return &StreamSession{
		client:   c,
		frame:    callbackFrame,
		streamID: streamID,
		log:      c.log,
	}
}

// ID 返回当前会话的 stream_id。
func (s *StreamSession) ID() string {
	return s.streamID
}

// Elapsed 返回从首次发送到现在经过的时间。未开始时返回 0。
func (s *StreamSession) Elapsed() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startTime.IsZero() {
		return 0
	}
	return time.Since(s.startTime)
}

// Remaining 返回距离 10 分钟超时还剩多少时间。未开始时返回 StreamMaxDuration。
func (s *StreamSession) Remaining() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.startTime.IsZero() {
		return StreamMaxDuration
	}
	remaining := StreamMaxDuration - time.Since(s.startTime)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsExpired 返回流式消息是否已超过 10 分钟限制。
func (s *StreamSession) IsExpired() bool {
	return s.Remaining() == 0
}

// IsFinished 返回流式消息是否已结束。
func (s *StreamSession) IsFinished() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.finished
}

// Update 发送流式消息的中间更新（finish=false）。
// 如果已超过 10 分钟限制，返回 ErrStreamExpired。
// 接近超时（最后 30 秒）时会输出警告日志。
func (s *StreamSession) Update(content string) error {
	return s.send(content, false)
}

// Finish 发送流式消息的最终内容并结束（finish=true）。
// 如果已超过 10 分钟限制，返回 ErrStreamExpired。
func (s *StreamSession) Finish(content string) error {
	return s.send(content, true)
}

func (s *StreamSession) send(content string, finish bool) error {
	s.mu.Lock()
	if s.finished {
		s.mu.Unlock()
		return fmt.Errorf("wecom-aibot: 流式消息已结束，不能再次发送")
	}

	// 首次发送时记录开始时间
	if s.startTime.IsZero() {
		s.startTime = time.Now()
	}

	// 检查是否超时
	elapsed := time.Since(s.startTime)
	if elapsed >= StreamMaxDuration {
		s.mu.Unlock()
		return ErrStreamExpired
	}

	// 接近超时时警告
	remaining := StreamMaxDuration - elapsed
	if remaining <= StreamWarnThreshold && !finish {
		s.log.Warn("流式消息即将超时（剩余 %v），请尽快调用 Finish()", remaining)
	}

	if finish {
		s.finished = true
	}
	s.mu.Unlock()

	return s.client.ReplyStream(s.frame, s.streamID, content, finish)
}
