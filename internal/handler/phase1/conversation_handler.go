package phase1handler

import (
	"agent_study/internal/logic/phase1"
	"agent_study/internal/model"
	"agent_study/internal/resp"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ConversationRegister 注册对话相关API
func ConversationRegister(apiGroup *gin.RouterGroup) {
	resp.HandlerWrapper(apiGroup, "conversation",
		[]*resp.Handler{
			resp.NewJsonHandler(handleChat),
			resp.NewHandler(http.MethodPost, "/chat/stream", handleChatStream),
			resp.NewJsonHandler(handleListConversations),
			resp.NewJsonHandler(handleGetConversation),
		})
}

// ChatReq 单次问答请求参数
type ChatReq struct {
	PromptID uint   `json:"prompt_id"`                   // 选择的Prompt ID，0表示不使用prompt
	Question string `json:"question" binding:"required"` // 用户问题
	Model    string `json:"model"`                       // 使用的模型，默认为kimi-k2.5
}

// ConversationListResp 对话列表响应
type ConversationListResp struct {
	List     []*model.Conversation `json:"list"`      // 对话列表
	Total    int64                 `json:"total"`     // 总数量
	Page     int                   `json:"page"`      // 当前页码
	PageSize int                   `json:"page_size"` // 每页数量
}

// handleChat
//
//	@Summary		单次问答
//	@Description	选择Prompt进行单次问答，返回回复和token统计
//	@Tags			conversation
//	@Accept			json
//	@Produce		json
//	@Param			body	body		ChatReq	true	"问答请求"
//	@Router			/conversation/chat [post]
//	@Success		200	{object}	resp.Result{data=phase1logic.ChatResponse}
func handleChat() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodPost, "/chat", func(c *gin.Context) (any, error) {
		var req ChatReq
		if err := c.ShouldBindJSON(&req); err != nil {
			return nil, err
		}

		// 调用业务逻辑
		chatResp, err := phase1logic.CreateChat(phase1logic.ChatRequest{
			PromptID: req.PromptID,
			Question: req.Question,
			Model:    req.Model,
		})
		if err != nil {
			return nil, err
		}

		return chatResp, nil
	}, nil
}

// handleChatStream
//
//	@Summary		流式问答
//	@Description	选择Prompt进行流式问答，通过SSE返回流式内容和统计信息
//	@Tags			conversation
//	@Accept			json
//	@Produce		text/event-stream
//	@Param			body	body	ChatReq	true	"问答请求"
//	@Router			/conversation/chat/stream [post]
//	@Success		200	{object}	phase1logic.ChatStreamResponse
func handleChatStream(c *gin.Context) {
	var req ChatReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置SSE响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 调用业务逻辑，通过回调函数发送流式内容
	finalResp, err := phase1logic.CreateChatStream(c.Request.Context(), phase1logic.ChatRequest{
		PromptID: req.PromptID,
		Question: req.Question,
		Model:    req.Model,
	}, func(chunk string) bool {
		// 发送流式内容片段
		chunkResp := phase1logic.ChatStreamResponse{
			Chunk: chunk,
			Done:  false,
		}
		c.SSEvent("message", chunkResp)
		c.Writer.Flush()
		return true
	})

	if err != nil {
		// 发送错误信息
		c.SSEvent("error", gin.H{"error": err.Error()})
		c.Writer.Flush()
		return
	}

	// 发送最终统计信息
	c.SSEvent("message", finalResp)
	c.Writer.Flush()
}

// handleListConversations
//
//	@Summary		获取对话列表
//	@Description	分页获取对话历史列表
//	@Tags			conversation
//	@Produce		json
//	@Param			page		query		int	false	"页码，默认1"
//	@Param			page_size	query		int	false	"每页数量，默认10，最大100"
//	@Router			/conversation/ [get]
//	@Success		200	{object}	resp.Result{data=ConversationListResp}
func handleListConversations() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodGet, "/", func(c *gin.Context) (any, error) {
		// 解析分页参数
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

		// 调用业务逻辑
		list, total, err := phase1logic.ListConversations(page, pageSize)
		if err != nil {
			return nil, err
		}

		return ConversationListResp{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}, nil
}

// handleGetConversation
//
//	@Summary		获取对话详情
//	@Description	根据ID获取对话详情，包含prompt信息
//	@Tags			conversation
//	@Produce		json
//	@Param			id	path		int	true	"对话ID"
//	@Router			/conversation/{id} [get]
//	@Success		200	{object}	resp.Result{data=model.Conversation}
func handleGetConversation() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodGet, "/:id", func(c *gin.Context) (any, error) {
		// 解析ID参数
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			return nil, err
		}

		// 调用业务逻辑
		return phase1logic.GetConversationByID(uint(id))
	}, nil
}
