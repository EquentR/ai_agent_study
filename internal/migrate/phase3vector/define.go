package phase3vectormigrate

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

// to002 文档向量数据表
var to002 = migrate.NewMigration("0.0.2", func(tx *gorm.DB) error {
	return nil
})
