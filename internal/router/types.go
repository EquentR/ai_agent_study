package router

import (
	"github.com/gin-gonic/gin"
	gStatic "github.com/soulteary/gin-static"
)

type Register func(apiGroup *gin.RouterGroup)

func RegisterAPI(apiGroup *gin.RouterGroup, registers []Register) {
	for _, register := range registers {
		register(apiGroup)
	}
}

func InitRouter(e *gin.Engine, registers []Register, baseUrl string, staticPath string) {
	// 注册 phase1 路由
	rg := e.Group(baseUrl)
	RegisterAPI(rg, registers)

	// 注册静态资源
	if staticPath != "" {
		e.Use(gStatic.Serve("/", gStatic.LocalFile(staticPath, true)))
	}
}
