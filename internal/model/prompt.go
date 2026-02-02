package model

import "time"

// Prompt LLM Prompt结构体
// 用于存储Prompt的基本信息
type Prompt struct {
	ID        uint      `json:"id" gorm:"type:integer;not null;primaryKey;autoIncrement;comment:主键ID"`
	Name      string    `json:"name" gorm:"type:varchar(255);not null;comment:Prompt名称"`
	Content   string    `json:"content" gorm:"type:text;not null;comment:Prompt内容"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;not null;comment:创建时间"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;not null;comment:更新时间"`
}

// TableName 指定表名
func (Prompt) TableName() string {
	return "prompts"
}

// PromptRating Prompt评分结构体
// 用于记录不同场景下Prompt的评分
type PromptRating struct {
	ID        uint      `json:"id" gorm:"type:integer;not null;primaryKey;autoIncrement;comment:主键ID"`
	PromptID  uint      `json:"prompt_id" gorm:"type:integer;not null;index;comment:关联的Prompt ID"`
	SceneName string    `json:"scene_name" gorm:"type:varchar(255);not null;comment:场景名称"`
	Score     float32   `json:"score" gorm:"type:real;not null;comment:评分(0-10)"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;not null;comment:创建时间"`
}

// TableName 指定表名
func (PromptRating) TableName() string {
	return "prompt_ratings"
}

// PromptRatingSummary Prompt评分汇总结构体
// 用于返回按场景分类的平均分
type PromptRatingSummary struct {
	SceneName string  `json:"scene_name"` // 场景名称
	AvgScore  float32 `json:"avg_score"`  // 平均分
	Count     int     `json:"count"`      // 评分次数
}
