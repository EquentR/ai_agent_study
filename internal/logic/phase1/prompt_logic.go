package phase1logic

import (
	"agent_study/internal/db"
	"agent_study/internal/model"
	"errors"

	"gorm.io/gorm"
)

// 错误定义
var (
	ErrPromptNotFound  = errors.New("prompt not found")
	ErrInvalidScore    = errors.New("score must be between 0 and 10")
	ErrInvalidPage     = errors.New("page must be greater than 0")
	ErrInvalidPageSize = errors.New("page size must be between 1 and 100")
	ErrEmptyName       = errors.New("prompt name cannot be empty")
	ErrEmptyContent    = errors.New("prompt content cannot be empty")
	ErrEmptySceneName  = errors.New("scene name cannot be empty")
)

// CreatePrompt 创建Prompt
// 参数:
//   - prompt: Prompt对象指针
//
// 返回:
//   - error: 错误信息
func CreatePrompt(prompt *model.Prompt) error {
	// 参数校验
	if prompt.Name == "" {
		return ErrEmptyName
	}
	if prompt.Content == "" {
		return ErrEmptyContent
	}

	return db.DB().Create(prompt).Error
}

// GetPromptByID 根据ID获取Prompt
// 参数:
//   - id: Prompt ID
//
// 返回:
//   - *model.Prompt: Prompt对象指针
//   - error: 错误信息
func GetPromptByID(id uint) (*model.Prompt, error) {
	var prompt model.Prompt
	err := db.DB().First(&prompt, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPromptNotFound
		}
		return nil, err
	}
	return &prompt, nil
}

// ListPrompts 分页查询Prompt列表
// 参数:
//   - page: 页码，从1开始
//   - pageSize: 每页数量，范围1-100
//
// 返回:
//   - []*model.Prompt: Prompt列表
//   - int64: 总数量
//   - error: 错误信息
func ListPrompts(page, pageSize int) ([]*model.Prompt, int64, error) {
	// 参数校验
	if page < 1 {
		return nil, 0, ErrInvalidPage
	}
	if pageSize < 1 || pageSize > 100 {
		return nil, 0, ErrInvalidPageSize
	}

	var prompts []*model.Prompt
	var total int64

	// 查询总数
	if err := db.DB().Model(&model.Prompt{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := db.DB().Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&prompts).Error; err != nil {
		return nil, 0, err
	}

	return prompts, total, nil
}

// UpdatePrompt 更新Prompt
// 参数:
//   - prompt: Prompt对象指针，必须包含ID
//
// 返回:
//   - error: 错误信息
func UpdatePrompt(prompt *model.Prompt) error {
	// 参数校验
	if prompt.Name == "" {
		return ErrEmptyName
	}
	if prompt.Content == "" {
		return ErrEmptyContent
	}

	// 检查是否存在
	var existing model.Prompt
	if err := db.DB().First(&existing, prompt.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPromptNotFound
		}
		return err
	}

	// 更新记录
	return db.DB().Model(&existing).Updates(map[string]interface{}{
		"name":    prompt.Name,
		"content": prompt.Content,
	}).Error
}

// DeletePrompt 删除Prompt
// 参数:
//   - id: Prompt ID
//
// 返回:
//   - error: 错误信息
func DeletePrompt(id uint) error {
	// 使用事务删除Prompt及其关联的评分记录
	return db.DB().Transaction(func(tx *gorm.DB) error {
		// 检查是否存在
		var prompt model.Prompt
		if err := tx.First(&prompt, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPromptNotFound
			}
			return err
		}

		// 删除关联的评分记录
		if err := tx.Where("prompt_id = ?", id).Delete(&model.PromptRating{}).Error; err != nil {
			return err
		}

		// 删除Prompt
		return tx.Delete(&prompt).Error
	})
}

// AddPromptRating 添加Prompt评分
// 参数:
//   - promptID: Prompt ID
//   - sceneName: 场景名称
//   - score: 评分(0-10)
//
// 返回:
//   - error: 错误信息
func AddPromptRating(promptID uint, sceneName string, score float32, conversationID *uint) error {
	// 参数校验
	if sceneName == "" {
		return ErrEmptySceneName
	}
	if score < 0 || score > 10 {
		return ErrInvalidScore
	}

	// 使用事务确保数据一致性
	return db.DB().Transaction(func(tx *gorm.DB) error {
		// 检查Prompt是否存在
		var prompt model.Prompt
		if err := tx.First(&prompt, promptID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrPromptNotFound
			}
			return err
		}

		if conversationID != nil {
			var conversation model.Conversation
			err := tx.First(&conversation, *conversationID).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return errors.New("conversation not found")
				}
				return err
			}
			if conversation.PromptID != promptID {
				return errors.New("conversation does not match prompt")
			}
		}

		// 创建评分记录
		rating := &model.PromptRating{
			PromptID:       promptID,
			SceneName:      sceneName,
			Score:          score,
			ConversationID: conversationID,
		}
		return tx.Create(rating).Error
	})
}

// GetPromptRatingSummary 获取Prompt评分汇总
// 按场景分类计算该Prompt在不同场景下的平均分
// 参数:
//   - promptID: Prompt ID
//
// 返回:
//   - []model.PromptRatingSummary: 评分汇总列表
//   - error: 错误信息
func GetPromptRatingSummary(promptID uint) ([]model.PromptRatingSummary, error) {
	// 检查Prompt是否存在
	var prompt model.Prompt
	if err := db.DB().First(&prompt, promptID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPromptNotFound
		}
		return nil, err
	}

	// 按场景分组查询平均分
	var summaries []model.PromptRatingSummary
	err := db.DB().Model(&model.PromptRating{}).
		Select("scene_name, AVG(score) as avg_score, COUNT(*) as count").
		Where("prompt_id = ?", promptID).
		Group("scene_name").
		Order("scene_name").
		Scan(&summaries).Error

	if err != nil {
		return nil, err
	}

	return summaries, nil
}

// ListPromptRatings 查询Prompt评分详情列表
// 支持根据PromptID、场景名称、对话ID过滤，支持分页
// 参数:
//   - promptID: Prompt ID，指针类型，传nil表示不过滤
//   - sceneName: 场景名称，精确匹配
//   - conversationID: 对话ID，指针类型，传nil表示不过滤
//   - page: 页码，从1开始
//   - pageSize: 每页数量，范围1-100
//
// 返回:
//   - []*model.PromptRatingDetail: 评分详情列表
//   - int64: 总数量
//   - error: 错误信息
func ListPromptRatings(promptID *uint, sceneName string, conversationID *uint, page, pageSize int) ([]*model.PromptRatingDetail, int64, error) {
	if page < 1 {
		return nil, 0, ErrInvalidPage
	}
	if pageSize < 1 || pageSize > 100 {
		return nil, 0, ErrInvalidPageSize
	}

	baseQuery := db.DB().Table("prompt_ratings").
		Joins("left join prompts on prompts.id = prompt_ratings.prompt_id")

	if promptID != nil && *promptID > 0 {
		baseQuery = baseQuery.Where("prompt_ratings.prompt_id = ?", *promptID)
	}
	if sceneName != "" {
		baseQuery = baseQuery.Where("prompt_ratings.scene_name = ?", sceneName)
	}
	if conversationID != nil && *conversationID > 0 {
		baseQuery = baseQuery.Where("prompt_ratings.conversation_id = ?", *conversationID)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var details []*model.PromptRatingDetail
	offset := (page - 1) * pageSize
	if err := baseQuery.Select("prompt_ratings.id, prompt_ratings.prompt_id, prompts.name as prompt_name, prompt_ratings.scene_name, prompt_ratings.score, prompt_ratings.conversation_id, prompt_ratings.created_at").
		Order("prompt_ratings.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&details).Error; err != nil {
		return nil, 0, err
	}

	return details, total, nil
}
