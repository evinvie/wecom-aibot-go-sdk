// Package wecomaibot provides a Go SDK for WeCom (企业微信) AI Bot WebSocket long connection.
package wecomaibot

import "encoding/json"

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

// Command constants define all WebSocket frame command types.
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
// Message & Event Types
// ---------------------------------------------------------------------------

// MsgType represents the type of a message payload.
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

// EventType represents the type of an event payload.
type EventType string

const (
	EventEnterChat     EventType = "enter_chat"
	EventTemplateCard  EventType = "template_card_event"
	EventFeedback      EventType = "feedback_event"
	EventDisconnected  EventType = "disconnected_event"
)

// ChatType represents the conversation type.
type ChatType string

const (
	ChatTypeSingle ChatType = "single"
	ChatTypeGroup  ChatType = "group"
)

// ChatTypeInt is the numeric chat type used in proactive messages.
type ChatTypeInt int

const (
	ChatTypeIntSingle ChatTypeInt = 1
	ChatTypeIntGroup  ChatTypeInt = 2
)

// ---------------------------------------------------------------------------
// Frame (the universal WebSocket message envelope)
// ---------------------------------------------------------------------------

// Frame is the top-level JSON structure for all WebSocket communication.
type Frame struct {
	Cmd     string          `json:"cmd,omitempty"`
	Headers Headers         `json:"headers"`
	Body    json.RawMessage `json:"body,omitempty"`
	ErrCode int             `json:"errcode,omitempty"`
	ErrMsg  string          `json:"errmsg,omitempty"`
}

// Headers carries per-frame metadata.
type Headers struct {
	ReqID string `json:"req_id"`
}

// ---------------------------------------------------------------------------
// Callback Bodies (server → client)
// ---------------------------------------------------------------------------

// MsgCallbackBody is the body of an aibot_msg_callback frame.
type MsgCallbackBody struct {
	MsgID    string          `json:"msgid"`
	AiBotID  string          `json:"aibotid"`
	ChatID   string          `json:"chatid,omitempty"`
	ChatType ChatType        `json:"chattype"`
	From     Sender          `json:"from"`
	MsgType  MsgType         `json:"msgtype"`
	Text     *TextContent    `json:"text,omitempty"`
	Image    *MediaContent   `json:"image,omitempty"`
	Mixed    *MixedContent   `json:"mixed,omitempty"`
	Voice    *VoiceContent   `json:"voice,omitempty"`
	File     *MediaContent   `json:"file,omitempty"`
	Video    *MediaContent   `json:"video,omitempty"`
}

// EventCallbackBody is the body of an aibot_event_callback frame.
type EventCallbackBody struct {
	MsgID      string    `json:"msgid"`
	CreateTime int64     `json:"create_time"`
	AiBotID    string    `json:"aibotid"`
	ChatID     string    `json:"chatid,omitempty"`
	From       Sender    `json:"from"`
	MsgType    MsgType   `json:"msgtype"`
	Event      EventInfo `json:"event"`
}

// Sender identifies the user who triggered the callback.
type Sender struct {
	UserID string `json:"userid"`
}

// EventInfo contains the event-specific details.
type EventInfo struct {
	EventType   EventType `json:"eventtype"`
	TaskID      string    `json:"task_id,omitempty"`
	OptionID    string    `json:"option_id,omitempty"`
	ButtonKey   string    `json:"button_key,omitempty"`
	FeedbackVal string    `json:"feedback_val,omitempty"`
}

// ---------------------------------------------------------------------------
// Content Types
// ---------------------------------------------------------------------------

// TextContent carries a plain text payload.
type TextContent struct {
	Content string `json:"content"`
}

// MediaContent carries a downloadable media reference.
type MediaContent struct {
	URL    string `json:"url,omitempty"`
	AESKey string `json:"aeskey,omitempty"`
	// MediaID is used when sending media that was uploaded via the upload API.
	MediaID string `json:"media_id,omitempty"`
}

// MixedContent carries a mixed (rich text + images) message.
type MixedContent struct {
	Items []MixedItem `json:"items"`
}

// MixedItem is one element in a mixed content message.
type MixedItem struct {
	MsgType MsgType       `json:"msgtype"`
	Text    *TextContent   `json:"text,omitempty"`
	Image   *MediaContent  `json:"image,omitempty"`
}

// VoiceContent carries a voice message (converted to text).
type VoiceContent struct {
	Content string `json:"content,omitempty"`
	URL     string `json:"url,omitempty"`
	AESKey  string `json:"aeskey,omitempty"`
}

// StreamContent defines a streaming message payload.
type StreamContent struct {
	ID      string `json:"id"`
	Finish  bool   `json:"finish"`
	Content string `json:"content"`
}

// MarkdownContent carries a Markdown payload.
type MarkdownContent struct {
	Content string `json:"content"`
}

// ---------------------------------------------------------------------------
// Template Card Types
// ---------------------------------------------------------------------------

// TemplateCard is a rich interactive card message.
type TemplateCard struct {
	CardType        string           `json:"card_type"`
	MainTitle       *CardTitle       `json:"main_title,omitempty"`
	SubTitleText    string           `json:"sub_title_text,omitempty"`
	HorizontalList  []CardKV         `json:"horizontal_content_list,omitempty"`
	ButtonList      []CardButton     `json:"button_list,omitempty"`
	TaskID          string           `json:"task_id,omitempty"`
}

// CardTitle is the main title block.
type CardTitle struct {
	Title string `json:"title"`
	Desc  string `json:"desc,omitempty"`
}

// CardKV is a horizontal key-value item.
type CardKV struct {
	KeyName string `json:"keyname"`
	Value   string `json:"value,omitempty"`
}

// CardButton is an interactive button.
type CardButton struct {
	Text  string `json:"text"`
	Style int    `json:"style,omitempty"`
	Key   string `json:"key"`
}

// ---------------------------------------------------------------------------
// Reply / Send Bodies (client → server)
// ---------------------------------------------------------------------------

// ReplyBody is used as the body for aibot_respond_msg.
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

// SendMsgBody is the body for aibot_send_msg (proactive push).
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

// UpdateCardBody is the body for aibot_respond_update_msg.
type UpdateCardBody struct {
	ResponseType string        `json:"response_type"`
	TemplateCard *TemplateCard `json:"template_card"`
}

// ---------------------------------------------------------------------------
// Media Upload Types
// ---------------------------------------------------------------------------

// UploadInitBody is the body for aibot_upload_media_init.
type UploadInitBody struct {
	Type        string `json:"type"`
	Filename    string `json:"filename"`
	TotalSize   int64  `json:"total_size"`
	TotalChunks int    `json:"total_chunks"`
	MD5         string `json:"md5"`
}

// UploadInitResp is the response body from upload init.
type UploadInitResp struct {
	UploadID string `json:"upload_id"`
}

// UploadChunkBody is the body for aibot_upload_media_chunk.
type UploadChunkBody struct {
	UploadID   string `json:"upload_id"`
	ChunkIndex int    `json:"chunk_index"`
	Base64Data string `json:"base64_data"`
}

// UploadFinishBody is the body for aibot_upload_media_finish.
type UploadFinishBody struct {
	UploadID string `json:"upload_id"`
}

// UploadFinishResp is the response body from upload finish.
type UploadFinishResp struct {
	MediaID string `json:"media_id"`
}
