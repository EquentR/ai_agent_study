package main

import (
	"agent_study/internal/db"
	"agent_study/internal/log"
	"agent_study/pkg/rag/fts5"
	"os"
	"strings"

	"gorm.io/gorm"
)

func main() {
	// 初始化日志
	log.Init(&log.Config{
		Level: "info",
	})
	db.Init(&db.Database{
		Name: "fts5_example",
		Params: []string{
			"_pragma=journal_mode(WAL)",
			"_pragma=busy_timeout(30000)",
			"_txlock=exclusive",
		},
		AutoCreate: true,
		InMemory:   false,
		DbDir:      SqliteDir,
		LogLevel:   "info",
	})
	if err := fts5.Init(db.DB()); err != nil {
		log.Fatalf("failed to initialize fts5: %v", err)
	}
	// 准备数据
	if err := prepare(db.DB()); err != nil {
		log.Fatalf("failed to prepare data: %v", err)
	}
	log.Info("done")
	sqlDB, _ := db.DB().DB()
	sqlDB.Close()
}

type Doc struct {
	Title   string
	Content string
}

func prepare(g *gorm.DB) error {
	// 读取数据，cmd/phase_3/1_simple_rag/docs.md，每个分割线一段落，第一行是标题，剩下是内容
	docs, err := readDocs("cmd/phase_3/1_simple_rag/docs.md")
	if err != nil {
		return err
	}

	// 插入数据到 SQLite
	for _, doc := range docs {
		if err := fts5.InsertDoc(g, doc.Title, doc.Content); err != nil {
			return err
		}
	}
	return nil
}

func readDocs(s string) ([]Doc, error) {
	// 读取文件内容，按分割线 --- 分割成多个段落，每个段落第一行是标题，剩下是内容
	file, err := os.ReadFile(s)
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(file), "---")
	docs := make([]Doc, 0, len(parts))
	for _, part := range parts {
		// 去掉收尾的空白字符
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		lines := strings.SplitN(part, "\n", 2)
		title := strings.Trim(strings.TrimSpace(lines[0]), "**")
		var content string
		if len(lines) > 1 {
			content = strings.TrimSpace(lines[1])
		}
		docs = append(docs, Doc{
			Title:   title,
			Content: content,
		})
	}
	return docs, nil
}
