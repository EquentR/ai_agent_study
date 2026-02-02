package phase1logic

import (
	"agent_study/internal/db"
	"agent_study/internal/model"
	"agent_study/pkg/llm_core/client/openai"
	llmModel "agent_study/pkg/llm_core/model"
	"context"
	"errors"
	"os"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 错误定义
var (
	ErrEmptyQuestion = errors.New("question cannot be empty")
	ErrLLMCallFailed = errors.New("failed to call LLM")
)

// ChatRequest 单次问答请求
type ChatRequest struct {
	PromptID uint   `json:"prompt_id"` // 选择的Prompt ID，0表示不使用prompt
	Question string `json:"question"`  // 用户问题
	Model    string `json:"model"`     // 使用的模型，默认为kimi-k2.5
}

// ChatResponse 单次问答响应
type ChatResponse struct {
	ConversationID   uint   `json:"conversation_id"`   // 对话ID
	Reply            string `json:"reply"`             // AI回复
	PromptTokens     int64  `json:"prompt_tokens"`     // Prompt Token数
	CompletionTokens int64  `json:"completion_tokens"` // Completion Token数
	TotalTokens      int64  `json:"total_tokens"`      // 总Token数
	Latency          int64  `json:"latency"`           // 响应延迟(毫秒)
}

// CreateChat 创建单次问答对话
// 参数:
//   - req: 问答请求
//
// 返回:
//   - *ChatResponse: 问答响应
//   - error: 错误信息
func CreateChat(req ChatRequest) (*ChatResponse, error) {
	// 参数校验
	if req.Question == "" {
		return nil, ErrEmptyQuestion
	}

	// 设置默认模型
	if req.Model == "" {
		req.Model = "kimi-k2.5"
	}

	// 构建消息列表
	messages := []llmModel.Message{
		{Role: llmModel.MessageUser, Content: req.Question},
	}

	// 如果指定了Prompt，添加system message
	var promptID uint = 0
	if req.PromptID > 0 {
		prompt, err := GetPromptByID(req.PromptID)
		if err != nil {
			return nil, err
		}
		promptID = prompt.ID
		// system message放在user message之后（根据phase_0示例）
		messages = append(messages, llmModel.Message{
			Role:    llmModel.MessageSystem,
			Content: prompt.Content,
		})
	}

	// 调用LLM
	llmClient := openai.NewOpenAiClient(
		os.Getenv("OPENAI_BASE_URL"),
		os.Getenv("OPENAI_API_KEY"),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	chatReq := llmModel.ChatRequest{
		Model:     req.Model,
		Messages:  messages,
		MaxTokens: 2048,
		TraceID:   uuid.New().String(),
	}

	resp, err := llmClient.Chat(ctx, chatReq)
	if err != nil {
		return nil, ErrLLMCallFailed
	}

	// 保存对话记录到数据库
	conversation := &model.Conversation{
		PromptID:         promptID,
		UserQuestion:     req.Question,
		AssistantReply:   resp.Content,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
		Latency:          resp.Latency.Milliseconds(),
		Model:            req.Model,
	}

	if err := db.DB().Create(conversation).Error; err != nil {
		return nil, err
	}

	// 返回响应
	return &ChatResponse{
		ConversationID:   conversation.ID,
		Reply:            resp.Content,
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
		Latency:          resp.Latency.Milliseconds(),
	}, nil
}

// ListConversations 分页查询对话列表
// 参数:
//   - page: 页码，从1开始
//   - pageSize: 每页数量，范围1-100
//
// 返回:
//   - []*model.Conversation: 对话列表
//   - int64: 总数量
//   - error: 错误信息
func ListConversations(page, pageSize int) ([]*model.Conversation, int64, error) {
	// 参数校验
	if page < 1 {
		return nil, 0, ErrInvalidPage
	}
	if pageSize < 1 || pageSize > 100 {
		return nil, 0, ErrInvalidPageSize
	}

	var conversations []*model.Conversation
	var total int64

	// 查询总数
	if err := db.DB().Model(&model.Conversation{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，预加载Prompt信息
	offset := (page - 1) * pageSize
	if err := db.DB().
		Preload("Prompt").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&conversations).Error; err != nil {
		return nil, 0, err
	}

	return conversations, total, nil
}

// GetConversationByID 根据ID获取对话详情
// 参数:
//   - id: 对话ID
//
// 返回:
//   - *model.Conversation: 对话对象指针
//   - error: 错误信息
func GetConversationByID(id uint) (*model.Conversation, error) {
	var conversation model.Conversation
	err := db.DB().Preload("Prompt").First(&conversation, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("conversation not found")
		}
		return nil, err
	}
	return &conversation, nil
}
