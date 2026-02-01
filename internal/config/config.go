package config

import (
	"agent_study/internal/db"
	"agent_study/internal/log"
)

type Config struct {
	Server Server      `yaml:"server"`
	Sqlite db.Database `yaml:"sqlite"`
	Log    log.Config  `yaml:"log"`
}

type Server struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	ApiBasePath string `yaml:"apiBasePath"`
	StaticPath  string `yaml:"staticPath"`
}
