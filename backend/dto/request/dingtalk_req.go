/**
 * @Author: Nan
 * @Date: 2025/2/14 15:49
 */

package request

// MessageType 消息类型枚举
type MessageType string

const (
	TextType       MessageType = "text"
	ImageType      MessageType = "image"
	LinkType       MessageType = "link"
	FileType       MessageType = "file"
	VoiceType      MessageType = "voice"
	OAType         MessageType = "oa"
	MarkdownType   MessageType = "markdown"
	ActionCardType MessageType = "action_card"
)

// DingTalkMessageReq txt消息请求
type DingTalkMessageReq struct {
	AgentId    string      `json:"agent_id,omitempty"`
	UseridList string      `json:"userid_list,omitempty"`
	DeptIdList string      `json:"dept_id_list,omitempty"`
	ToAllUser  bool        `json:"to_all_user,omitempty"`
	MsgType    MessageType `json:"msg_type" binding:"required"`                                                                                                                                   // 消息类型
	Msg        interface{} `json:"msg" binding:"required" swaggertype:"object" oneOf:"TextContent,ImageContent,LinkContent,FileContent,VoiceContent,OAContent,MarkdownContent,ActionCardContent"` // 消息内容，根据类型不同而不同
}

// TextContent 文本消息内容
type TextContent struct {
	Content string `json:"content"`
}

// ImageContent 图片消息内容
type ImageContent struct {
	MediaId string `json:"media_id"`
}

// LinkContent 链接消息内容
type LinkContent struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	MessageUrl string `json:"message_url"`
	PicUrl     string `json:"pic_url"`
}

// FileContent 文件消息内容
type FileContent struct {
	MediaId string `json:"media_id"`
}

// VoiceContent 语音消息内容
type VoiceContent struct {
	MediaId  string `json:"media_id"`
	Duration string `json:"duration"`
}

// OAContent OA消息内容
type OAContent struct {
	MsgUrl string `json:"msg_url"`
	Head   struct {
		BgColor string `json:"bgcolor"`
		Text    string `json:"text"`
	} `json:"head"`
	Body struct {
		Author    string `json:"author"`
		FileCount string `json:"file_count"`
		Image     string `json:"image"`
		Content   string `json:"content"`
		Rich      struct {
			Unit string `json:"unit"`
			Num  string `json:"num"`
		} `json:"rich"`
		Form struct {
			Value string `json:"value"`
			Key   string `json:"key"`
		} `json:"form"`
		Title string `json:"title"`
	} `json:"body"`
	StatusBar struct {
		StatusValue string `json:"status_value"`
		StatusBg    string `json:"status_bg"`
	} `json:"status_bar"`
}

// MarkdownContent Markdown消息内容
type MarkdownContent struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// ActionCardContent ActionCard消息内容
type ActionCardContent struct {
	Title          string `json:"title"`
	Markdown       string `json:"markdown"`
	SingleTitle    string `json:"single_title"`
	SingleURL      string `json:"single_url"`
	BtnOrientation string `json:"btn_orientation"`
	BtnJsonList    string `json:"btn_json_list"`
}
