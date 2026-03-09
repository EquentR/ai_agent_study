package main

import (
	"agent_study/internal/app/phase_3_vector_rag"
	"agent_study/internal/config"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	// 读取配置文件，初始化 LLM 客户端
	f, err := os.ReadFile("conf/phase3vector/app.yaml")
	if err != nil {
		panic(err)
	}

	cfg := &config.Config{}
	err = yaml.Unmarshal(f, cfg)
	if err != nil {
		panic(err)
	}

	// 启动服务
	phase_3_vector_rag.Serve(cfg)
}
