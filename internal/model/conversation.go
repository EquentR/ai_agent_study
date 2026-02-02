package model

import "time"

// Conversation 对话记录结构体
// 用于存储单次问答的对话信息
type Conversation struct {
	ID               uint      `json:"id" gorm:"type:integer;not null;primaryKey;autoIncrement;comment:主键ID"`
	PromptID         uint      `json:"prompt_id" gorm:"type:integer;not null;index;comment:使用的Prompt ID"`
	UserQuestion     string    `json:"user_question" gorm:"type:text;not null;comment:用户问题"`
	AssistantReply   string    `json:"assistant_reply" gorm:"type:text;not null;comment:AI回复"`
	PromptTokens     int64     `json:"prompt_tokens" gorm:"type:integer;not null;comment:Prompt Token数"`
	CompletionTokens int64     `json:"completion_tokens" gorm:"type:integer;not null;comment:Completion Token数"`
	TotalTokens      int64     `json:"total_tokens" gorm:"type:integer;not null;comment:总Token数"`
	Latency          int64     `json:"latency" gorm:"type:integer;not null;comment:响应延迟(毫秒)"`
	Model            string    `json:"model" gorm:"type:varchar(100);not null;comment:使用的模型"`
	CreatedAt        time.Time `json:"created_at" gorm:"type:datetime;not null;comment:创建时间"`
	TraceId          string    `json:"trace_id" gorm:"type:text;not null;comment:调用追踪ID"`
	// 关联的Prompt对象(用于查询时返回)
	Prompt *Prompt `json:"prompt,omitempty" gorm:"foreignKey:PromptID"`
}

// TableName 指定表名
func (Conversation) TableName() string {
	return "conversations"
}
