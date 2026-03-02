@echo off
REM 测试 MCP STDIO Agent 示例

echo === MCP STDIO Agent 测试 ===
echo.

REM 检查环境变量
if "%OPENAI_BASE_URL%"=="" (
    echo 警告: OPENAI_BASE_URL 未设置
    echo 请设置环境变量：
    echo   set OPENAI_BASE_URL=your_api_base_url
    echo   set OPENAI_API_KEY=your_api_key
    exit /b 1
)

REM 编译 server
echo 1. 编译 Server...
cd server
go build -o server.exe main.go
if %errorlevel% neq 0 (
    echo Server 编译失败
    exit /b 1
)
cd ..

REM 编译 agent
echo 2. 编译 Agent...
cd agent
go build -o agent.exe main.go
if %errorlevel% neq 0 (
    echo Agent 编译失败
    exit /b 1
)

echo 3. 运行 Agent...
echo.

REM 运行测试
agent.exe -server "../server/server.exe" -question "请帮我生成一个UUID"

cd ..\..
