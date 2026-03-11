package model

import "time"

// LongTermMemory 用于存储单个用户跨会话压缩后的长期记忆。
type LongTermMemory struct {
	ID        uint      `json:"id" gorm:"type:integer;not null;primaryKey;autoIncrement;comment:主键ID"`
	Username  string    `json:"username" gorm:"type:varchar(128);not null;uniqueIndex;comment:用户名"`
	Summary   string    `json:"summary" gorm:"type:text;not null;default:'';comment:长期记忆摘要"`
	CreatedAt time.Time `json:"created_at" gorm:"type:datetime;not null;comment:创建时间"`
	UpdatedAt time.Time `json:"updated_at" gorm:"type:datetime;not null;comment:更新时间"`
}

// TableName 显式固定表名，避免依赖 GORM 默认复数化规则带来的不确定性。
func (LongTermMemory) TableName() string {
	return "long_term_memories"
}
