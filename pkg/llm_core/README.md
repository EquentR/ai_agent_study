# LLM Core
提供辅助工具，包括 token 计数器（支持本地快速计数和精确 tiktoken 计数）。

## 附件能力

`model.Message` 新增可选字段 `Attachments`，支持将图片和文本文件作为附件发送给 LLM。

- 图片附件：会按 OpenAI 兼容格式转换为 `image_url`（data URL）
- 文本附件：会作为文本片段追加到消息内容中

不传附件时，原有文本消息调用方式保持不变。

### tools

定义 LLM 交互所需的数据结构、接口和类型。
### model

提供不同 LLM 服务商的客户端实现（目前支持 OpenAI 兼容接口）。
### client

## 子包说明

LLM 核心功能库，提供与大语言模型交互的基础能力。

