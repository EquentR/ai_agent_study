package phase1migrate

import (
	"agent_study/internal/migrate"
	"agent_study/internal/model"

	"gorm.io/gorm"
)

// to001 初始化迁移，创建数据版本表
var to001 = migrate.NewMigration("0.0.1", func(tx *gorm.DB) error {
	err := tx.AutoMigrate(&model.DataVersion{})
	if err != nil {
		return err
	}
	return nil
})

// to002 Prompt功能迁移，创建prompts表和prompt_ratings表
var to002 = migrate.NewMigration("0.0.2", func(tx *gorm.DB) error {
	// 创建prompts表
	if err := tx.AutoMigrate(&model.Prompt{}); err != nil {
		return err
	}
	// 创建prompt_ratings表
	if err := tx.AutoMigrate(&model.PromptRating{}); err != nil {
		return err
	}
	return nil
})

// to003 对话功能迁移，创建conversations表
var to003 = migrate.NewMigration("0.0.3", func(tx *gorm.DB) error {
	// 创建conversations表
	if err := tx.AutoMigrate(&model.Conversation{}); err != nil {
		return err
	}
	return nil
})
