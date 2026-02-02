package phase1router

import (
	"agent_study/internal/handler"
	phase1handler "agent_study/internal/handler/phase1"

	"github.com/gin-gonic/gin"
	gStatic "github.com/soulteary/gin-static"
)

type Register func(apiGroup *gin.RouterGroup)

var (
	registers = []Register{
		handler.Register,
		phase1handler.PromptRegister,
		phase1handler.ConversationRegister,
	}
)

func registerAPI(apiGroup *gin.RouterGroup) {
	for _, register := range registers {
		register(apiGroup)
	}
}

func InitRouter(e *gin.Engine, baseUrl string, staticPath string) {
	// 注册 phase1 路由
	rg := e.Group(baseUrl)
	registerAPI(rg)

	// 注册静态资源
	if staticPath != "" {
		e.Use(gStatic.Serve("/", gStatic.LocalFile(staticPath, true)))
	}
}
