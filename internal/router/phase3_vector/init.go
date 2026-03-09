package phase3_vector_router

import (
	"agent_study/internal/handler"
	"agent_study/internal/router"

	"github.com/gin-gonic/gin"
)

var (
	registers = []router.Register{
		handler.Register,
	}
)

func InitRouter(e *gin.Engine, baseUrl string, staticPath string) {
	router.InitRouter(e, registers, baseUrl, staticPath)
}
