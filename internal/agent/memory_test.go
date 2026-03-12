package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"agent_study/internal/model"
	llmModel "agent_study/pkg/llm_core/model"
	toolTypes "agent_study/pkg/types"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMemoryManagerShortTermBufferKeepsOrderAndReturnsCopy(t *testing.T) {
	mgr, err := NewMemoryManager(MemoryOptions{})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	mgr.AddMessage(llmModel.Message{Role: llmModel.RoleUser, Content: "hello"})
	mgr.AddMessage(llmModel.Message{Role: llmModel.RoleAssistant, Content: "hi"})

	got := mgr.ShortTermMessages()
	if len(got) != 2 {
		t.Fatalf("ShortTermMessages() len = %d, want 2", len(got))
	}
	if got[0].Content != "hello" || got[1].Content != "hi" {
		t.Fatalf("ShortTermMessages() = %#v, want ordered session messages", got)
	}

	got[0].Content = "mutated"
	again := mgr.ShortTermMessages()
	if again[0].Content != "hello" {
		t.Fatalf("ShortTermMessages() should return a defensive copy, got %#v", again)
	}
}

func TestMemoryManagerLongTermCreatesDefaultUserRecord(t *testing.T) {
	db := newTestMemoryDB(t)

	mgr, err := NewMemoryManager(MemoryOptions{DB: db})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	summary, err := mgr.LongTermSummary(context.Background())
	if err != nil {
		t.Fatalf("LongTermSummary() error = %v", err)
	}
	if summary != "" {
		t.Fatalf("LongTermSummary() = %q, want empty summary for first load", summary)
	}

	var stored model.LongTermMemory
	if err := db.Where("username = ?", "default").First(&stored).Error; err != nil {
		t.Fatalf("default long-term memory record not created: %v", err)
	}
}

func TestMemoryManagerShortTermBufferDeepCopiesToolCallThoughtSignature(t *testing.T) {
	mgr, err := NewMemoryManager(MemoryOptions{})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	mgr.AddMessage(llmModel.Message{
		Role: llmModel.RoleAssistant,
		ToolCalls: []toolTypes.ToolCall{{
			Name:             "search",
			ThoughtSignature: []byte("abc"),
		}},
	})

	got := mgr.ShortTermMessages()
	got[0].ToolCalls[0].ThoughtSignature[0] = 'z'

	again := mgr.ShortTermMessages()
	if string(again[0].ToolCalls[0].ThoughtSignature) != "abc" {
		t.Fatalf("ThoughtSignature should be deep-copied, got %q", string(again[0].ToolCalls[0].ThoughtSignature))
	}
}

func TestMemoryManagerShortTermBufferDeepCopiesReasoningItems(t *testing.T) {
	mgr, err := NewMemoryManager(MemoryOptions{})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	mgr.AddMessage(llmModel.Message{
		Role: llmModel.RoleAssistant,
		ReasoningItems: []llmModel.ReasoningItem{{
			ID: "rs_1",
			Summary: []llmModel.ReasoningSummary{{
				Text: "plan first",
			}},
		}},
	})

	got := mgr.ShortTermMessages()
	got[0].ReasoningItems[0].Summary[0].Text = "mutated"

	again := mgr.ShortTermMessages()
	if again[0].ReasoningItems[0].Summary[0].Text != "plan first" {
		t.Fatalf("ReasoningItems should be deep-copied, got %#v", again[0].ReasoningItems)
	}
}

func TestNewMemoryManagerAutoMigratesLongTermMemorySchema(t *testing.T) {
	db := newBareTestMemoryDB(t)

	if _, err := NewMemoryManager(MemoryOptions{DB: db}); err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}
	if !db.Migrator().HasTable(&model.LongTermMemory{}) {
		t.Fatalf("NewMemoryManager() should auto-migrate long_term_memories")
	}
}

func TestNewMemoryManagerDoesNotOverwriteExistingDataVersion(t *testing.T) {
	db := newBareTestMemoryDB(t)
	if err := db.AutoMigrate(&model.DataVersion{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	record := &model.DataVersion{ID: 1, Version: "9.9.9"}
	if err := db.Create(record).Error; err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := NewMemoryManager(MemoryOptions{DB: db}); err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	var saved model.DataVersion
	if err := db.First(&saved, 1).Error; err != nil {
		t.Fatalf("First() error = %v", err)
	}
	if saved.Version != "9.9.9" {
		t.Fatalf("data version = %q, want %q", saved.Version, "9.9.9")
	}
}

func TestMemoryManagerFlushShortTermToLongTermPersistsCompressedSummary(t *testing.T) {
	db := newTestMemoryDB(t)

	mgr, err := NewMemoryManager(MemoryOptions{
		DB: db,
		Compressor: func(ctx context.Context, currentSummary string, session []llmModel.Message) (string, error) {
			parts := make([]string, 0, len(session)+1)
			if currentSummary != "" {
				parts = append(parts, currentSummary)
			}
			for _, message := range session {
				parts = append(parts, message.Role+":"+message.Content)
			}
			return strings.Join(parts, " | "), nil
		},
	})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	mgr.AddMessage(llmModel.Message{Role: llmModel.RoleUser, Content: "user likes golang"})
	mgr.AddMessage(llmModel.Message{Role: llmModel.RoleAssistant, Content: "remembered"})

	got, err := mgr.FlushShortTermToLongTerm(context.Background())
	if err != nil {
		t.Fatalf("FlushShortTermToLongTerm() error = %v", err)
	}

	want := "user:user likes golang | assistant:remembered"
	if got != want {
		t.Fatalf("FlushShortTermToLongTerm() = %q, want %q", got, want)
	}
	if len(mgr.ShortTermMessages()) != 0 {
		t.Fatalf("FlushShortTermToLongTerm() should clear short-term buffer")
	}

	var stored model.LongTermMemory
	if err := db.Where("username = ?", "default").First(&stored).Error; err != nil {
		t.Fatalf("stored long-term memory query error = %v", err)
	}
	if stored.Summary != want {
		t.Fatalf("stored summary = %q, want %q", stored.Summary, want)
	}
}

func TestMemoryManagerDefaultCompressorKeepsValidUTF8(t *testing.T) {
	db := newTestMemoryDB(t)

	mgr, err := NewMemoryManager(MemoryOptions{DB: db, MaxSummaryChars: 5})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	mgr.AddMessage(llmModel.Message{Role: llmModel.RoleUser, Content: "你好世界你好世界"})

	summary, err := mgr.FlushShortTermToLongTerm(context.Background())
	if err != nil {
		t.Fatalf("FlushShortTermToLongTerm() error = %v", err)
	}
	if !utf8.ValidString(summary) {
		t.Fatalf("summary should stay valid UTF-8, got %q", summary)
	}
}

func newTestMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := newBareTestMemoryDB(t)
	if err := db.AutoMigrate(&model.LongTermMemory{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	return db
}

func newBareTestMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	return db
}
