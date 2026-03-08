package fts5

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSearchDocsSupportsSpaceSeparatedKeywords(t *testing.T) {
	db := openTestDB(t)

	if err := Init(db); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	if err := InsertDoc(db, "猫鼠队观察", "所谓“猫鼠队”，指的是家里的两个阵营。上大分，则是指它们达成了默契配合。"); err != nil {
		t.Fatalf("InsertDoc() error = %v", err)
	}
	if err := InsertDoc(db, "无关内容", "这是一篇完全不相关的文档。"); err != nil {
		t.Fatalf("InsertDoc() error = %v", err)
	}

	singleDocs, err := SearchDocs(db, "猫鼠队", 5)
	if err != nil {
		t.Fatalf("SearchDocs() single keyword error = %v", err)
	}
	if len(singleDocs) != 1 {
		t.Fatalf("len(singleDocs) = %d, want 1", len(singleDocs))
	}

	docs, err := SearchDocs(db, "猫鼠队 上大分", 5)
	if err != nil {
		t.Fatalf("SearchDocs() error = %v", err)
	}
	if len(docs) != 1 {
		t.Fatalf("len(docs) = %d, want 1", len(docs))
	}
	if got := docs[0]["title"]; got != "猫鼠队观察" {
		t.Fatalf("docs[0][title] = %v, want %q", got, "猫鼠队观察")
	}
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}

	return db
}
