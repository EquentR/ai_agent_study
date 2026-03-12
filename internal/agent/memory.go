package agent

import (
	"context"
	"errors"
	"strings"
	"sync"

	phase4migrate "agent_study/internal/migrate/phase4"
	"agent_study/internal/model"
	llmModel "agent_study/pkg/llm_core/model"
	toolTypes "agent_study/pkg/types"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	defaultMemoryUsername = "default"
	defaultSummaryChars   = 2000
	ellipsis              = "..."
)

var ErrLongTermMemoryDisabled = errors.New("long-term memory is not configured")

// MemoryCompressor 用于把已有摘要和当前短期会话压缩成最终持久化的长期记忆摘要。
type MemoryCompressor func(ctx context.Context, currentSummary string, session []llmModel.Message) (string, error)

// MemoryOptions 控制记忆能力是仅保存在进程内，还是额外持久化到配置的数据库中。
type MemoryOptions struct {
	DB              *gorm.DB
	Username        string
	Compressor      MemoryCompressor
	MaxSummaryChars int
}

// MemoryManager 负责维护短期消息的防御性拷贝，并在启用数据库时把压缩后的历史
// 同步到长期存储。
type MemoryManager struct {
	mu              sync.RWMutex
	shortTerm       []llmModel.Message
	compressor      MemoryCompressor
	maxSummaryChars int
	longTerm        *longTermMemoryStore
}

type longTermMemoryStore struct {
	db       *gorm.DB
	username string
}

// NewMemoryManager 会先补齐默认配置，只有在显式传入数据库句柄时才启用长期记忆。
func NewMemoryManager(options MemoryOptions) (*MemoryManager, error) {
	maxSummaryChars := options.MaxSummaryChars
	if maxSummaryChars <= 0 {
		maxSummaryChars = defaultSummaryChars
	}

	compressor := options.Compressor
	if compressor == nil {
		compressor = newSimpleMemoryCompressor(maxSummaryChars)
	}

	manager := &MemoryManager{
		compressor:      compressor,
		maxSummaryChars: maxSummaryChars,
	}

	if options.DB != nil {
		// MemoryManager 只依赖长期记忆表，因此这里走一个更窄的初始化路径，避免
		// 为了单个能力把整套应用迁移流程都执行一遍。
		if err := phase4migrate.BootstrapWithDB(options.DB, phase4migrate.CurrentVersion); err != nil {
			return nil, err
		}

		store := &longTermMemoryStore{
			db:       options.DB,
			username: normalizeUsername(options.Username),
		}
		// 先为当前用户补一条空记录，保证后续读取和刷入长期记忆时，无论是否已经
		// 产生摘要，都能依赖这条稳定存在的记录。
		if _, err := store.getOrCreate(); err != nil {
			return nil, err
		}
		manager.longTerm = store
	}

	return manager, nil
}

func (m *MemoryManager) AddMessage(message llmModel.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.shortTerm = append(m.shortTerm, cloneMessage(message))
}

func (m *MemoryManager) ShortTermMessages() []llmModel.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return cloneMessages(m.shortTerm)
}

func (m *MemoryManager) ClearShortTerm() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.shortTerm = nil
}

// LongTermSummary 返回当前持久化的长期摘要，同时不把底层存储实现暴露出去。
func (m *MemoryManager) LongTermSummary(ctx context.Context) (string, error) {
	if m.longTerm == nil {
		return "", ErrLongTermMemoryDisabled
	}

	record, err := m.longTerm.getOrCreateWithContext(ctx)
	if err != nil {
		return "", err
	}
	return record.Summary, nil
}

// FlushShortTermToLongTerm 会把当前短期会话压缩进长期摘要，并且只在保存成功后
// 才清空短期缓存。
func (m *MemoryManager) FlushShortTermToLongTerm(ctx context.Context) (string, error) {
	if m.longTerm == nil {
		return "", ErrLongTermMemoryDisabled
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	record, err := m.longTerm.getOrCreateWithContext(ctx)
	if err != nil {
		return "", err
	}
	// 没有新增短期消息时就不必调用压缩器；这类调用通常只是想拿到当前已持久化的
	// 摘要内容。
	if len(m.shortTerm) == 0 {
		return record.Summary, nil
	}

	// 传给压缩器的是克隆后的快照，避免自定义压缩器通过共享切片意外修改
	// 管理器内部维护的短期会话状态。
	summary, err := m.compressor(ctx, record.Summary, cloneMessages(m.shortTerm))
	if err != nil {
		return "", err
	}
	if err := m.longTerm.saveSummary(ctx, summary); err != nil {
		return "", err
	}

	m.shortTerm = nil
	return summary, nil
}

func normalizeUsername(username string) string {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return defaultMemoryUsername
	}
	return trimmed
}

func newSimpleMemoryCompressor(maxChars int) MemoryCompressor {
	return func(_ context.Context, currentSummary string, session []llmModel.Message) (string, error) {
		parts := make([]string, 0, len(session)+1)
		// 先把已有摘要放进去，保证多次 flush 会持续累积上下文，而不是每次都整段
		// 覆盖掉更早的长期记忆。
		if text := compactWhitespace(currentSummary); text != "" {
			parts = append(parts, text)
		}
		for _, message := range session {
			if text := compactWhitespace(formatMemoryMessage(message)); text != "" {
				parts = append(parts, text)
			}
		}
		return limitSummary(strings.Join(parts, "\n"), maxChars), nil
	}
}

func formatMemoryMessage(message llmModel.Message) string {
	content := compactWhitespace(message.Content)
	if len(message.ToolCalls) > 0 {
		names := make([]string, 0, len(message.ToolCalls))
		for _, toolCall := range message.ToolCalls {
			if toolCall.Name == "" {
				continue
			}
			names = append(names, toolCall.Name)
		}
		if len(names) > 0 {
			// 即使 assistant 消息没有文本内容，也要保留调用过哪些工具，因为这段执行历史
			// 本身就可能成为后续会话需要参考的上下文。
			if content == "" {
				content = "tool_calls=" + strings.Join(names, ",")
			} else {
				content = content + " tool_calls=" + strings.Join(names, ",")
			}
		}
	}
	if message.Role == llmModel.RoleTool && message.ToolCallId != "" {
		// 工具响应在被压缩后，原始 assistant 发起调用的那条消息可能已经不在了，
		// 因此这里把 tool_call_id 一并写进文本，保留可追溯关系。
		if content == "" {
			content = "tool_call_id=" + message.ToolCallId
		} else {
			content = "tool_call_id=" + message.ToolCallId + " " + content
		}
	}
	if content == "" {
		return ""
	}
	role := compactWhitespace(message.Role)
	if role == "" {
		return content
	}
	return role + ": " + content
}

func compactWhitespace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func limitSummary(text string, maxChars int) string {
	runes := []rune(text)
	if maxChars <= 0 || len(runes) <= maxChars {
		return text
	}
	if maxChars <= len(ellipsis) {
		return string(runes[:maxChars])
	}

	// 按 rune 而不是按 byte 截断，避免摘要被裁剪后把多字节字符切坏，产生非法
	// UTF-8 文本。
	headChars := (maxChars - len(ellipsis)) / 2
	tailChars := maxChars - len(ellipsis) - headChars
	return string(runes[:headChars]) + ellipsis + string(runes[len(runes)-tailChars:])
}

func cloneMessages(messages []llmModel.Message) []llmModel.Message {
	if len(messages) == 0 {
		return nil
	}
	cloned := make([]llmModel.Message, 0, len(messages))
	for _, message := range messages {
		cloned = append(cloned, cloneMessage(message))
	}
	return cloned
}

func cloneMessage(message llmModel.Message) llmModel.Message {
	cloned := message

	if len(message.Attachments) > 0 {
		// 深拷贝附件字节数据，避免后续对原消息的内存修改反向污染已经存入短期记忆
		// 的快照内容。
		cloned.Attachments = make([]llmModel.Attachment, 0, len(message.Attachments))
		for _, attachment := range message.Attachments {
			clonedAttachment := attachment
			if len(attachment.Data) > 0 {
				clonedAttachment.Data = append([]byte(nil), attachment.Data...)
			}
			cloned.Attachments = append(cloned.Attachments, clonedAttachment)
		}
	}

	if len(message.ReasoningItems) > 0 {
		cloned.ReasoningItems = make([]llmModel.ReasoningItem, 0, len(message.ReasoningItems))
		for _, item := range message.ReasoningItems {
			clonedItem := item
			if len(item.Summary) > 0 {
				clonedItem.Summary = append([]llmModel.ReasoningSummary(nil), item.Summary...)
			}
			cloned.ReasoningItems = append(cloned.ReasoningItems, clonedItem)
		}
	}

	if len(message.ToolCalls) > 0 {
		// ToolCall 内部还带有字节切片字段（如 ThoughtSignature），如果只做浅拷贝，
		// 仍然会和原始消息共享底层内存。
		cloned.ToolCalls = make([]toolTypes.ToolCall, 0, len(message.ToolCalls))
		for _, toolCall := range message.ToolCalls {
			clonedToolCall := toolCall
			if len(toolCall.ThoughtSignature) > 0 {
				clonedToolCall.ThoughtSignature = append([]byte(nil), toolCall.ThoughtSignature...)
			}
			cloned.ToolCalls = append(cloned.ToolCalls, clonedToolCall)
		}
	}

	return cloned
}

func (s *longTermMemoryStore) getOrCreate() (*model.LongTermMemory, error) {
	return s.getOrCreateWithContext(context.Background())
}

func (s *longTermMemoryStore) getOrCreateWithContext(ctx context.Context) (*model.LongTermMemory, error) {
	seed := &model.LongTermMemory{Username: s.username, Summary: ""}
	// 先尝试插入，再通过冲突忽略吸收并发竞争，保证重复启动或并发请求最终都会
	// 安全收敛到同一条用户记录。
	if err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "username"}},
			DoNothing: true,
		}).
		Create(seed).Error; err != nil {
		return nil, err
	}

	// 插入并忽略冲突的尝试结束后再查一遍，保证调用方拿到的始终是数据库里的最终记录，
	// 不受“本次创建”还是“别的协程已提前创建”影响。
	record := &model.LongTermMemory{}
	if err := s.db.WithContext(ctx).Where("username = ?", s.username).First(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func (s *longTermMemoryStore) saveSummary(ctx context.Context, summary string) error {
	record, err := s.getOrCreateWithContext(ctx)
	if err != nil {
		return err
	}
	record.Summary = summary
	return s.db.WithContext(ctx).Save(record).Error
}
