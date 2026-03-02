#!/bin/bash

# 测试 MCP STDIO Agent 示例

echo "=== MCP STDIO Agent 测试 ==="
echo ""

# 设置环境变量（请根据实际情况修改）
if [ -z "$OPENAI_BASE_URL" ]; then
    echo "警告: OPENAI_BASE_URL 未设置"
    echo "请设置环境变量："
    echo "  export OPENAI_BASE_URL='your_api_base_url'"
    echo "  export OPENAI_API_KEY='your_api_key'"
    exit 1
fi

# 编译 server
echo "1. 编译 Server..."
cd server
go build -o server.exe main.go
if [ $? -ne 0 ]; then
    echo "Server 编译失败"
    exit 1
fi
cd ..

# 编译 agent
echo "2. 编译 Agent..."
cd agent
go build -o agent.exe main.go
if [ $? -ne 0 ]; then
    echo "Agent 编译失败"
    exit 1
fi

echo "3. 运行 Agent..."
echo ""

# 运行测试
./agent.exe -server "../server/server.exe" -question "请帮我生成一个UUID"

cd ../..
