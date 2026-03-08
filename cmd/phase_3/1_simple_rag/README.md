# Simple RAG 示例（SQLite FTS5）

这个示例演示一个最小可理解的本地 RAG 流程：

- 使用 `cmd/phase_3/1_simple_rag/docs.md` 作为本地知识库语料
- 把文档切分后写入 SQLite FTS5 虚拟表
- 通过 `search_fts5` 工具把全文检索能力暴露给 LLM
- 让模型在回答用户问题时结合工具结果生成最终答案

## 目录说明

- `main.go`：主示例入口，注册 `search_fts5` 和 `get_weather` 两个工具，驱动多轮 tool call
- `prepare.go`：读取 `docs.md`，按 `---` 分段后写入 SQLite FTS5 表
- `docs.md`：示例知识库，包含梗解释、娱乐内容和科普内容等文本

## 示例流程

1. `prepare.go` 调用 `readDocs` 读取 `docs.md`
2. 每段文本提取标题和正文后，通过 `fts5.InsertDoc` 写入 `fts5_docs`
3. `main.go` 启动对话，向模型声明 `search_fts5` 工具
4. 模型根据用户问题自主决定是否调用全文搜索
5. 程序执行搜索，把结果作为 `tool` 消息回传给模型
6. 模型基于检索结果和天气工具结果输出最终答案

## search_fts5 工具说明

`search_fts5(query: string)` 会调用 `pkg/rag/fts5.SearchDocs`。

当前支持：

- 单关键词搜索：例如 `提拉米苏`
- 多关键词搜索：例如 `猫鼠队 上大分`
- 多关键词通过空格分隔，底层会转换成 FTS5 的 `AND` 查询

这意味着：

- 输入 `猫鼠队 上大分` 时，会检索同时包含这两个关键词的文档
- 输入为空白字符串时，返回空结果而不是 SQL 报错

## 代码关注点

- 本示例把“检索”实现成工具调用，而不是在应用层先检索再拼 prompt，更方便观察 LLM 的自主决策过程
- `SystemPrompt` 明确要求模型在信息不足时优先查知识库，减少直接猜测
- `maxRounds` 用于限制工具调用轮数，避免无限循环
- `searchFTS5` 返回 JSON 数组，便于模型直接消费结构化结果

## 适合学习什么

- 本地知识库如何接入 LLM 工具调用链路
- SQLite FTS5 如何作为轻量级全文检索后端
- 一个最小 RAG 示例中“知识准备 -> 检索工具 -> 多轮问答”的基本结构

## 当前状态说明

这个目录目前更适合作为代码学习示例：

- `main.go` 和 `prepare.go` 都是 `package main`，且都定义了 `main()`
- 因此当前不能直接对整个目录执行 `go run ./cmd/phase_3/1_simple_rag`

如果后续要把它变成可直接运行的完整示例，建议把“数据准备”和“问答执行”拆成两个独立命令，或把公共逻辑抽到共享包中。

## 相关文档

- `pkg/rag/fts5/README.md`：FTS5 封装说明
- `README.md`：项目总览与学习路线
