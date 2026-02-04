package phase_1_prompt_call

import (
	"agent_study/internal/app"
	"agent_study/internal/config"
	"agent_study/internal/db"
	"agent_study/internal/log"
	phase1migrate "agent_study/internal/migrate/phase1"
	phase1router "agent_study/internal/router/phase1"
	"fmt"
	"net"

	"github.com/gin-gonic/gin"
)

func Serve(c *config.Config) {
	app.GracefulExit()
	// 初始化日志
	log.Init(&c.Log)

	// 初始化数据库
	db.Init(&c.Sqlite)

	// 迁移表结构
	phase1migrate.Bootstrap("0.0.4")

	// 初始化路由
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()

	phase1router.InitRouter(e, c.Server.ApiBasePath, c.Server.StaticPath)

	addr := fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panicf("Failed to listen on %s: %v", addr, err)
	}

	// 启动服务器
	go func() {
		if err := e.RunListener(ln); err != nil {
			log.Panicf("Failed to run server: %v", err)
		}
	}()
	log.Infof("gin listening on %s", addr)

	// 等待关闭信号
	select {
	case <-app.Ctx.Done():
		_ = ln.Close()
		log.Info("Shutting down server...")
	}
}
