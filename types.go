// Package wecomaibot 提供企业微信智能机器人 WebSocket 长连接 Go SDK。
package wecomaibot

import "encoding/json"

// ---------------------------------------------------------------------------
// 命令常量
// ---------------------------------------------------------------------------

// WebSocket 帧命令类型常量。
const (
	CmdSubscribe         = "aibot_subscribe"
	CmdPing              = "ping"
	CmdMsgCallback       = "aibot_msg_callback"
	CmdEventCallback     = "aibot_event_callback"
	CmdRespondMsg        = "aibot_respond_msg"
	CmdRespondWelcomeMsg = "aibot_respond_welcome_msg"
	CmdRespondUpdateMsg  = "aibot_respond_update_msg"
	CmdSendMsg           = "aibot_send_msg"
	CmdUploadMediaInit   = "aibot_upload_media_init"
	CmdUploadMediaChunk  = "aibot_upload_media_chunk"
	CmdUploadMediaFinish = "aibot_upload_media_finish"
)

// ---------------------------------------------------------------------------
// 消息与事件类型
// ---------------------------------------------------------------------------

// MsgType 表示消息负载的类型。
type MsgType string

const (
	MsgTypeText     MsgType = "text"
	MsgTypeImage    MsgType = "image"
	MsgTypeMixed    MsgType = "mixed"
	MsgTypeVoice    MsgType = "voice"
	MsgTypeFile     MsgType = "file"
	MsgTypeVideo    MsgType = "video"
	MsgTypeStream   MsgType = "stream"
	MsgTypeMarkdown MsgType = "markdown"
	MsgTypeEvent    MsgType = "event"
	MsgTypeCard     MsgType = "template_card"
)

// EventType 表示事件负载的类型。
type EventType string

const (
	EventEnterChat    EventType = "enter_chat"
	EventTemplateCard EventType = "template_card_event"
	EventFeedback     EventType = "feedback_event"
	EventDisconnected EventType = "disconnected_event"
)

// ChatType 表示会话类型（字符串形式）。
type ChatType string

const (
	ChatTypeSingle ChatType = "single" // 单聊
	ChatTypeGroup  ChatType = "group"  // 群聊
)

// ChatTypeInt 是主动推送消息时使用的数字会话类型。
type ChatTypeInt int

const (
	ChatTypeIntSingle ChatTypeInt = 1 // 单聊
	ChatTypeIntGroup  ChatTypeInt = 2 // 群聊
)

// ---------------------------------------------------------------------------
// Frame（WebSocket 通信的统一消息信封）
// ---------------------------------------------------------------------------

// Frame 是所有 WebSocket 通信的顶层 JSON 结构。
type Frame struct {
	Cmd     string          `json:"cmd,omitempty"`
	Headers Headers         `json:"headers"`
	Body    json.RawMessage `json:"body,omitempty"`
	ErrCode int             `json:"errcode,omitempty"`
	ErrMsg  string          `json:"errmsg,omitempty"`
}

// Headers 携带每帧的元数据。
type Headers struct {
	ReqID string `json:"req_id"`
}

// ---------------------------------------------------------------------------
// 回调消息体（服务端 → 客户端）
// ---------------------------------------------------------------------------

// MsgCallbackBody 是 aibot_msg_callback 帧的消息体。
type MsgCallbackBody struct {
	MsgID    string        `json:"msgid"`
	AiBotID  string        `json:"aibotid"`
	ChatID   string        `json:"chatid,omitempty"`
	ChatType ChatType      `json:"chattype"`
	From     Sender        `json:"from"`
	MsgType  MsgType       `json:"msgtype"`
	Text     *TextContent  `json:"text,omitempty"`
	Image    *MediaContent `json:"image,omitempty"`
	Mixed    *MixedContent `json:"mixed,omitempty"`
	Voice    *VoiceContent `json:"voice,omitempty"`
	File     *MediaContent `json:"file,omitempty"`
	Video    *MediaContent `json:"video,omitempty"`
}

// EventCallbackBody 是 aibot_event_callback 帧的消息体。
type EventCallbackBody struct {
	MsgID      string    `json:"msgid"`
	CreateTime int64     `json:"create_time"`
	AiBotID    string    `json:"aibotid"`
	ChatID     string    `json:"chatid,omitempty"`
	From       Sender    `json:"from"`
	MsgType    MsgType   `json:"msgtype"`
	Event      EventInfo `json:"event"`
}

// Sender 标识触发回调的用户。
type Sender struct {
	UserID string `json:"userid"`
}

// EventInfo 包含事件的详细信息。
type EventInfo struct {
	EventType   EventType `json:"eventtype"`
	TaskID      string    `json:"task_id,omitempty"`
	OptionID    string    `json:"option_id,omitempty"`
	ButtonKey   string    `json:"button_key,omitempty"`
	FeedbackVal string    `json:"feedback_val,omitempty"`
}

// ---------------------------------------------------------------------------
// 内容类型
// ---------------------------------------------------------------------------

// TextContent 承载纯文本内容。
type TextContent struct {
	Content string `json:"content"`
}

// MediaContent 承载可下载的媒体资源引用。
type MediaContent struct {
	URL     string `json:"url,omitempty"`
	AESKey  string `json:"aeskey,omitempty"`
	MediaID string `json:"media_id,omitempty"` // 通过上传接口获得的素材 ID
}

// MixedContent 承载图文混排消息。
type MixedContent struct {
	Items []MixedItem `json:"items"`
}

// MixedItem 是图文混排消息中的一个元素。
type MixedItem struct {
	MsgType MsgType       `json:"msgtype"`
	Text    *TextContent  `json:"text,omitempty"`
	Image   *MediaContent `json:"image,omitempty"`
}

// VoiceContent 承载语音消息（已转为文字）。
type VoiceContent struct {
	Content string `json:"content,omitempty"`
	URL     string `json:"url,omitempty"`
	AESKey  string `json:"aeskey,omitempty"`
}

// StreamContent 定义流式消息的负载。
type StreamContent struct {
	ID      string `json:"id"`
	Finish  bool   `json:"finish"`
	Content string `json:"content"`
}

// MarkdownContent 承载 Markdown 格式的内容。
type MarkdownContent struct {
	Content string `json:"content"`
}

// ---------------------------------------------------------------------------
// 模板卡片类型
// ---------------------------------------------------------------------------

// TemplateCard 是富交互卡片消息。
type TemplateCard struct {
	CardType       string       `json:"card_type"`
	MainTitle      *CardTitle   `json:"main_title,omitempty"`
	SubTitleText   string       `json:"sub_title_text,omitempty"`
	HorizontalList []CardKV    `json:"horizontal_content_list,omitempty"`
	ButtonList     []CardButton `json:"button_list,omitempty"`
	TaskID         string       `json:"task_id,omitempty"`
}

// CardTitle 是卡片的主标题区块。
type CardTitle struct {
	Title string `json:"title"`
	Desc  string `json:"desc,omitempty"`
}

// CardKV 是水平键值对条目。
type CardKV struct {
	KeyName string `json:"keyname"`
	Value   string `json:"value,omitempty"`
}

// CardButton 是可交互的按钮。
type CardButton struct {
	Text  string `json:"text"`
	Style int    `json:"style,omitempty"`
	Key   string `json:"key"`
}

// ---------------------------------------------------------------------------
// 回复/发送消息体（客户端 → 服务端）
// ---------------------------------------------------------------------------

// ReplyBody 是 aibot_respond_msg 的消息体。
type ReplyBody struct {
	MsgType      MsgType          `json:"msgtype"`
	Text         *TextContent     `json:"text,omitempty"`
	Stream       *StreamContent   `json:"stream,omitempty"`
	Markdown     *MarkdownContent `json:"markdown,omitempty"`
	Image        *MediaContent    `json:"image,omitempty"`
	Voice        *MediaContent    `json:"voice,omitempty"`
	Video        *MediaContent    `json:"video,omitempty"`
	File         *MediaContent    `json:"file,omitempty"`
	TemplateCard *TemplateCard    `json:"template_card,omitempty"`
}

// SendMsgBody 是 aibot_send_msg（主动推送）的消息体。
type SendMsgBody struct {
	ChatID       string           `json:"chatid"`
	ChatType     ChatTypeInt      `json:"chat_type"`
	MsgType      MsgType          `json:"msgtype"`
	Text         *TextContent     `json:"text,omitempty"`
	Markdown     *MarkdownContent `json:"markdown,omitempty"`
	Image        *MediaContent    `json:"image,omitempty"`
	Voice        *MediaContent    `json:"voice,omitempty"`
	Video        *MediaContent    `json:"video,omitempty"`
	File         *MediaContent    `json:"file,omitempty"`
	TemplateCard *TemplateCard    `json:"template_card,omitempty"`
}

// UpdateCardBody 是 aibot_respond_update_msg 的消息体。
type UpdateCardBody struct {
	ResponseType string        `json:"response_type"`
	TemplateCard *TemplateCard `json:"template_card"`
}

// ---------------------------------------------------------------------------
// 素材上传类型
// ---------------------------------------------------------------------------

// UploadInitBody 是 aibot_upload_media_init 的请求体。
type UploadInitBody struct {
	Type        string `json:"type"`
	Filename    string `json:"filename"`
	TotalSize   int64  `json:"total_size"`
	TotalChunks int    `json:"total_chunks"`
	MD5         string `json:"md5"`
}

// UploadInitResp 是上传初始化的响应体。
type UploadInitResp struct {
	UploadID string `json:"upload_id"`
}

// UploadChunkBody 是 aibot_upload_media_chunk 的请求体。
type UploadChunkBody struct {
	UploadID   string `json:"upload_id"`
	ChunkIndex int    `json:"chunk_index"`
	Base64Data string `json:"base64_data"`
}

// UploadFinishBody 是 aibot_upload_media_finish 的请求体。
type UploadFinishBody struct {
	UploadID string `json:"upload_id"`
}

// UploadFinishResp 是上传完成的响应体。
type UploadFinishResp struct {
	MediaID string `json:"media_id"`
}
