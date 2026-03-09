package phase3vectormigrate

import (
	"agent_study/internal/log"
	"agent_study/internal/migrate"
)

var versionMigrations = []migrate.Migration{
	to001, to002,
}

func Bootstrap(version string) {
	log.Info("DB migration starting...")
	migrate.AutoMigrate(version, versionMigrations)
}
