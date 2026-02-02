package migrate

import (
	"agent_study/internal/db"
	"agent_study/internal/log"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Migration struct {
	Version string
	Fun     func()
}

func migrationInTransaction(v string, fun func(tx *gorm.DB) error) {
	err := db.DB().Transaction(func(tx *gorm.DB) error {
		SetDataVersion(tx, v)
		return fun(tx)
	})
	if err != nil {
		log.Fatal(fmt.Sprintf("%s auto migration failed", v), zap.Error(err))
	}
}

func NewMigration(v string, fun func(tx *gorm.DB) error) Migration {
	return Migration{
		Version: v,
		Fun: func() {
			migrationInTransaction(v, fun)
		},
	}
}
