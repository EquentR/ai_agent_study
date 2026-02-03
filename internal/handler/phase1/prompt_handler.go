package phase1handler

import (
	"agent_study/internal/logic/phase1"
	"agent_study/internal/model"
	"agent_study/internal/resp"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PromptRegister 注册Prompt相关API
func PromptRegister(apiGroup *gin.RouterGroup) {
	resp.HandlerWrapper(apiGroup, "prompt",
		[]*resp.Handler{
			resp.NewJsonHandler(handleListPrompts),
			resp.NewJsonHandler(handleGetPrompt),
			resp.NewJsonHandler(handleCreatePrompt),
			resp.NewJsonHandler(handleUpdatePrompt),
			resp.NewJsonHandler(handleDeletePrompt),
			resp.NewJsonHandler(handleAddRating),
			resp.NewJsonHandler(handleGetRatingSummary),
			resp.NewJsonHandler(handleListRatings),
		})
}

// PromptCreateReq 创建Prompt请求参数
type PromptCreateReq struct {
	Name    string `json:"name" binding:"required"`    // Prompt名称
	Content string `json:"content" binding:"required"` // Prompt内容
}

// PromptUpdateReq 更新Prompt请求参数
type PromptUpdateReq struct {
	Name    string `json:"name" binding:"required"`    // Prompt名称
	Content string `json:"content" binding:"required"` // Prompt内容
}

// PromptRatingReq 添加评分请求参数
type PromptRatingReq struct {
	SceneName      string  `json:"scene_name" binding:"required"` // 场景名称
	Score          float32 `json:"score" binding:"required"`      // 评分(0-10)
	ConversationID *uint   `json:"conversation_id,omitempty"`     // 关联对话ID
}

// PromptListResp 分页列表响应
type PromptListResp struct {
	List     []*model.Prompt `json:"list"`      // Prompt列表
	Total    int64           `json:"total"`     // 总数量
	Page     int             `json:"page"`      // 当前页码
	PageSize int             `json:"page_size"` // 每页数量
}

// PromptRatingListResp 评分明细列表响应
// 用于返回评分明细查询结果
type PromptRatingListResp struct {
	List     []*model.PromptRatingDetail `json:"list"`
	Total    int64                       `json:"total"`
	Page     int                         `json:"page"`
	PageSize int                         `json:"page_size"`
}

// handleListPrompts
//
//	@Summary		获取Prompt列表
//	@Description	分页获取Prompt列表
//	@Tags			prompt
//	@Produce		json
//	@Param			page		query		int	false	"页码，默认1"
//	@Param			page_size	query		int	false	"每页数量，默认10，最大100"
//	@Router			/prompt/ [get]
//	@Success		200	{object}	resp.Result{data=PromptListResp}
func handleListPrompts() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodGet, "/", func(c *gin.Context) (any, error) {
		// 解析分页参数
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

		// 调用业务逻辑
		list, total, err := phase1logic.ListPrompts(page, pageSize)
		if err != nil {
			return nil, err
		}

		return PromptListResp{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}, nil
}

// handleGetPrompt
//
//	@Summary		获取单个Prompt
//	@Description	根据ID获取Prompt详情
//	@Tags			prompt
//	@Produce		json
//	@Param			id	path		int	true	"Prompt ID"
//	@Router			/prompt/{id} [get]
//	@Success		200	{object}	resp.Result{data=model.Prompt}
func handleGetPrompt() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodGet, "/:id", func(c *gin.Context) (any, error) {
		// 解析ID参数
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			return nil, err
		}

		// 调用业务逻辑
		return phase1logic.GetPromptByID(uint(id))
	}, nil
}

// handleCreatePrompt
//
//	@Summary		创建Prompt
//	@Description	创建新的Prompt
//	@Tags			prompt
//	@Accept			json
//	@Produce		json
//	@Param			body	body		PromptCreateReq	true	"创建参数"
//	@Router			/prompt/ [post]
//	@Success		200	{object}	resp.Result{data=model.Prompt}
func handleCreatePrompt() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodPost, "/", func(c *gin.Context) (any, error) {
		var req PromptCreateReq
		if err := c.ShouldBindJSON(&req); err != nil {
			return nil, err
		}

		// 创建Prompt对象
		prompt := &model.Prompt{
			Name:    req.Name,
			Content: req.Content,
		}

		// 调用业务逻辑
		if err := phase1logic.CreatePrompt(prompt); err != nil {
			return nil, err
		}

		return prompt, nil
	}, nil
}

// handleUpdatePrompt
//
//	@Summary		更新Prompt
//	@Description	根据ID更新Prompt
//	@Tags			prompt
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"Prompt ID"
//	@Param			body	body		PromptUpdateReq	true	"更新参数"
//	@Router			/prompt/{id} [put]
//	@Success		200	{object}	resp.Result{data=model.Prompt}
func handleUpdatePrompt() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodPut, "/:id", func(c *gin.Context) (any, error) {
		// 解析ID参数
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			return nil, err
		}

		var req PromptUpdateReq
		if err := c.ShouldBindJSON(&req); err != nil {
			return nil, err
		}

		// 创建Prompt对象
		prompt := &model.Prompt{
			ID:      uint(id),
			Name:    req.Name,
			Content: req.Content,
		}

		// 调用业务逻辑
		if err := phase1logic.UpdatePrompt(prompt); err != nil {
			return nil, err
		}

		// 返回更新后的Prompt
		return phase1logic.GetPromptByID(uint(id))
	}, nil
}

// handleDeletePrompt
//
//	@Summary		删除Prompt
//	@Description	根据ID删除Prompt
//	@Tags			prompt
//	@Produce		json
//	@Param			id	path		int	true	"Prompt ID"
//	@Router			/prompt/{id} [delete]
//	@Success		200	{object}	resp.Result{data=string}
func handleDeletePrompt() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodDelete, "/:id", func(c *gin.Context) (any, error) {
		// 解析ID参数
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			return nil, err
		}

		// 调用业务逻辑
		if err := phase1logic.DeletePrompt(uint(id)); err != nil {
			return nil, err
		}

		return "deleted successfully", nil
	}, nil
}

// handleAddRating
//
//	@Summary		添加Prompt评分
//	@Description	为指定Prompt在某场景下添加评分
//	@Tags			prompt
//	@Accept			json
//	@Produce		json
//	@Param			id		path		int				true	"Prompt ID"
//	@Param			body	body		PromptRatingReq	true	"评分参数"
//	@Router			/prompt/{id}/rating [post]
//	@Success		200	{object}	resp.Result{data=string}
func handleAddRating() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodPost, "/:id/rating", func(c *gin.Context) (any, error) {
		// 解析ID参数
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			return nil, err
		}

		var req PromptRatingReq
		if err := c.ShouldBindJSON(&req); err != nil {
			return nil, err
		}

		// 调用业务逻辑
		if err := phase1logic.AddPromptRating(uint(id), req.SceneName, req.Score, req.ConversationID); err != nil {
			return nil, err
		}

		return "rating added successfully", nil
	}, nil
}

// handleGetRatingSummary
//
//	@Summary		获取Prompt评分汇总
//	@Description	按场景分类获取该Prompt的平均分
//	@Tags			prompt
//	@Produce		json
//	@Param			id	path		int	true	"Prompt ID"
//	@Router			/prompt/{id}/rating/summary [get]
//	@Success		200	{object}	resp.Result{data=[]model.PromptRatingSummary}
func handleGetRatingSummary() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodGet, "/:id/rating/summary", func(c *gin.Context) (any, error) {
		// 解析ID参数
		id, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			return nil, err
		}

		// 调用业务逻辑
		return phase1logic.GetPromptRatingSummary(uint(id))
	}, nil
}

// handleListRatings
//
//	@Summary		获取评分明细列表
//	@Description	支持按Prompt、场景和对话ID过滤评分明细
//	@Tags			prompt
//	@Produce		json
//	@Param			prompt_id		query		int		false	"Prompt ID(可选)"
//	@Param			scene_name		query		string	false	"场景名称(可选)"
//	@Param			conversation_id	query		int		false	"对话ID(可选)"
//	@Param			page			query		int		false	"页码，默认1"
//	@Param			page_size		query		int		false	"每页数量，默认10，最大100"
//	@Router			/prompt/rating/list [get]
//	@Success		200	{object}	resp.Result{data=PromptRatingListResp}
func handleListRatings() (string, string, resp.JsonResultWrapper, []resp.WrapperOption) {
	return http.MethodGet, "/rating/list", func(c *gin.Context) (any, error) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
		sceneName := c.Query("scene_name")

		var promptID *uint
		if promptIDRaw := c.Query("prompt_id"); promptIDRaw != "" {
			value, err := strconv.ParseUint(promptIDRaw, 10, 64)
			if err != nil {
				return nil, err
			}
			parsed := uint(value)
			promptID = &parsed
		}

		var conversationID *uint
		if conversationIDRaw := c.Query("conversation_id"); conversationIDRaw != "" {
			value, err := strconv.ParseUint(conversationIDRaw, 10, 64)
			if err != nil {
				return nil, err
			}
			parsed := uint(value)
			conversationID = &parsed
		}

		list, total, err := phase1logic.ListPromptRatings(promptID, sceneName, conversationID, page, pageSize)
		if err != nil {
			return nil, err
		}

		return PromptRatingListResp{
			List:     list,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}, nil
}
