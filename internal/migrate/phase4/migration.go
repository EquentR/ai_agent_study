package phase4migrate

import (
	"agent_study/internal/log"
	"agent_study/internal/migrate"
	"agent_study/internal/model"
	"errors"

	"gorm.io/gorm"
)

const CurrentVersion = "0.0.6"

var versionMigrations = []migrate.Migration{
	to001,
	to002,
}

func Bootstrap(version string) {
	log.Info("DB migration starting...")
	migrate.AutoMigrate(version, versionMigrations)
}

// BootstrapWithDB 是给 memory 这类组件使用的轻量初始化路径，它们只要求长期记忆表
// 在首次使用前存在即可。
func BootstrapWithDB(database *gorm.DB, _ string) error {
	if database == nil {
		return errors.New("database is nil")
	}

	// 这里刻意不走完整的版本迁移链，只让依赖方在已经拿到 gorm.DB 的前提下，
	// 以最小代价完成自己真正需要的表初始化。
	return database.AutoMigrate(&model.LongTermMemory{})
}
