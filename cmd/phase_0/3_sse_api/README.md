# SSE API 测试界面

## 功能说明
这是一个基于 Server-Sent Events (SSE) 的实时流式对话测试界面。

## 使用方法

### 1. 启动服务器
```bash
cd E:\Develop\AI\agent_study\cmd\phase_0\3_sse_api
go run main.go
```

### 2. 访问测试界面
在浏览器中打开: `http://localhost:8080`

### 3. 测试对话
- 在输入框中输入问题
- 点击"发送"按钮或按回车键
- 实时查看 AI 的流式响应
- 对话结束后会显示使用统计信息

## 技术特点

### 后端 (Go + Gin)
- **SSE 流式响应**: 使用 Server-Sent Events 实现实时数据推送
- **实时刷新**: 每个 chunk 发送后立即 flush，确保低延迟
- **统计信息**: 返回 token 使用量和响应时间信息
- **CORS 支持**: 允许跨域访问

### 前端 (HTML + JavaScript)
- **EventSource API**: 原生 SSE 客户端实现
- **流式显示**: 实时追加 AI 响应内容
- **美观界面**: 现代化渐变设计，响应式布局
- **状态管理**: 防止重复请求，按钮状态控制
- **自动滚动**: 消息自动滚动到底部

## 文件结构
```
3_sse_api/
├── main.go       # 后端服务器代码
├── index.html    # 前端测试界面
└── README.md     # 本文档
```

## API 端点

### GET /
返回 HTML 测试界面

### GET /chat?question={问题}
- **参数**: question - 用户问题
- **返回**: SSE 事件流
- **事件类型**: LLMResp
- **数据格式**: 
  ```json
  {
    "chunk": "响应文本片段",
    "usage": "统计信息（最后发送）"
  }
  ```

## 注意事项
- 确保 index.html 文件与 main.go 在同一目录
- 服务器默认运行在 8080 端口
- SSE 连接在完成后会自动关闭
- 发送新请求前会等待当前请求完成
