package migrate

import (
	"agent_study/internal/db"
	"agent_study/internal/log"
	"agent_study/internal/model"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func GetDataVersion() (string, error) {
	version := &model.DataVersion{
		ID:      1,
		Version: "0.0.0",
	}
	err := db.DB().FirstOrCreate(version).Error
	if err != nil {
		return "", err
	}
	return version.Version, nil
}

// SetDataVersion 设置数据版本，所有迭代过程一定要在事务中进行，并且该步骤一定要最先被调用
func SetDataVersion(tx *gorm.DB, version string) {
	if tx == nil {
		log.Fatal("数据库以及事务不能为空")
		return
	}
	err := tx.Model(&model.DataVersion{}).Where("id = ?", 1).
		Update("version", version).Error
	if err != nil {
		log.Fatal("设置数据版本失败", zap.Error(err))
	}
}
