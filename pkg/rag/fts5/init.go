package fts5

import (
	"fmt"
	"strings"

	"gorm.io/gorm"
)

const (
	DocTableName   = "fts5_docs"
	CreateTableSQL = `
CREATE VIRTUAL TABLE IF NOT EXISTS %s USING fts5(
	title,
	content
);`
)

func Init(db *gorm.DB) error {
	// 检查 FTS5 是否可用
	if err := db.Exec("CREATE VIRTUAL TABLE IF NOT EXISTS fts5_test USING fts5(content)").Error; err != nil {
		return err
	}
	// 删除测试表
	if err := db.Exec("DROP TABLE IF EXISTS fts5_test").Error; err != nil {
		return err
	}
	// 检查是否已经存在 FTS5 表
	var count int64
	if err := db.Raw("SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", DocTableName).Scan(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		// 创建 FTS5 表
		createSQL := fmt.Sprintf(CreateTableSQL, DocTableName)
		if err := db.Exec(createSQL).Error; err != nil {
			return err
		}
		return nil
	}
	return nil
}

func InsertDoc(db *gorm.DB, title string, content string) error {
	insertSQL := fmt.Sprintf("INSERT INTO %s (title, content) VALUES (?, ?)", DocTableName)
	return db.Exec(insertSQL, title, content).Error
}

func SearchDocs(db *gorm.DB, query string, topK int) ([]map[string]interface{}, error) {
	normalizedQuery := normalizeQuery(query)
	if normalizedQuery == "" {
		return []map[string]interface{}{}, nil
	}

	searchSQL := fmt.Sprintf("SELECT title, content FROM %s WHERE %s MATCH ? LIMIT ?", DocTableName, DocTableName)
	rows, err := db.Raw(searchSQL, normalizedQuery, topK).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var title, content string
		if err := rows.Scan(&title, &content); err != nil {
			return nil, err
		}
		result := map[string]interface{}{
			"title":   title,
			"content": content,
		}
		results = append(results, result)
	}
	return results, nil
}

func normalizeQuery(query string) string {
	terms := strings.Fields(strings.TrimSpace(query))
	if len(terms) == 0 {
		return ""
	}
	if len(terms) == 1 {
		return terms[0]
	}

	normalized := make([]string, 0, len(terms))
	for _, term := range terms {
		normalized = append(normalized, quoteTerm(term))
	}

	return strings.Join(normalized, " AND ")
}

func quoteTerm(term string) string {
	return `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
}

func DeleteDoc(db *gorm.DB, title string) error {
	deleteSQL := fmt.Sprintf("DELETE FROM %s WHERE title = ?", DocTableName)
	return db.Exec(deleteSQL, title).Error
}
