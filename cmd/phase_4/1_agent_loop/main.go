package main

import (
	"agent_study/internal/config"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	// 读取配置文件，初始化 LLM 客户端
	f, err := os.ReadFile("conf/phase4/app.yaml")
	if err != nil {
		panic(err)
	}

	expended := os.ExpandEnv(string(f))

	cfg := &config.Config{}
	err = yaml.Unmarshal([]byte(expended), cfg)
	if err != nil {
		panic(err)
	}
	// TODO 检查配置

	// TODO 启动交互式终端，接受用户输入并实现Agent运行

}
