# SQLite FTS5 封装

`pkg/rag/fts5` 提供一个非常轻量的 SQLite FTS5 封装，适合在本地 RAG、小型知识库检索或 demo 场景中直接使用。

当前封装聚焦四个能力：

- 初始化 FTS5 虚拟表
- 写入文档
- 删除文档
- 执行全文搜索

## 提供的 API

```go
func Init(db *gorm.DB) error
func InsertDoc(db *gorm.DB, title string, content string) error
func SearchDocs(db *gorm.DB, query string, topK int) ([]map[string]interface{}, error)
func DeleteDoc(db *gorm.DB, title string) error
```

## 数据结构

当前虚拟表名固定为 `fts5_docs`，包含两个字段：

- `title`
- `content`

建表语句位于 `pkg/rag/fts5/init.go` 中：

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS fts5_docs USING fts5(
  title,
  content
);
```

## 搜索行为

`SearchDocs` 在执行 SQL 前会先规范化查询串：

- 空查询：返回空结果
- 单关键词：保持原样查询
- 多关键词：按空白拆分，并转换为 `AND` 查询

例如：

- `提拉米苏` -> `提拉米苏`
- `猫鼠队 上大分` -> `"猫鼠队" AND "上大分"`

这样做的原因是 SQLite FTS5 对原始查询串有自己的查询语法。直接把空格分隔的文本原样传入时，常会被解释成短语匹配，导致“多个关键词都存在但没有连续出现”的文档无法命中。

## 使用示例

```go
if err := fts5.Init(db); err != nil {
    return err
}

if err := fts5.InsertDoc(db, "标题", "正文内容"); err != nil {
    return err
}

docs, err := fts5.SearchDocs(db, "猫鼠队 上大分", 5)
if err != nil {
    return err
}

for _, doc := range docs {
    fmt.Println(doc["title"], doc["content"])
}
```

## 适用场景

- 本地知识库 demo
- Agent 工具调用中的轻量检索后端
- 不依赖额外搜索服务的原型验证

## 当前限制

- 只封装了最基础的增删查，没有排序、打分、摘要高亮等能力
- 返回值为 `[]map[string]interface{}`，适合快速集成，但类型约束较弱
- 当前仅搜索 `title` 和 `content` 两列
- 未额外接入中文分词器，中文检索效果依赖 SQLite FTS5 默认行为和输入关键词质量

## 测试

当前包包含一个回归测试，用来验证“空格分隔的多个关键词”可以正常搜索：

```bash
go test ./pkg/rag/fts5 -v
```

## 相关示例

- `cmd/phase_3/1_simple_rag/README.md`：如何在简单 RAG 示例里使用这个包
