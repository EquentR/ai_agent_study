package phase4migrate

import (
	"agent_study/internal/migrate"
	"agent_study/internal/model"

	"gorm.io/gorm"
)

// to001 先确保全局共用的数据版本表存在，再继续 phase 4 自己的持久化表迁移。
var to001 = migrate.NewMigration("0.0.5", func(tx *gorm.DB) error {
	return tx.AutoMigrate(&model.DataVersion{})
})

// to002 引入 MemoryManager 依赖的按用户维度长期记忆表。
var to002 = migrate.NewMigration("0.0.6", func(tx *gorm.DB) error {
	return tx.AutoMigrate(&model.LongTermMemory{})
})
