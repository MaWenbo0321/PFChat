package main

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const sessionOperationLockCount = 64

var errActiveSessionExists = errors.New("active session already exists")

// 同一会话的消息生成、逐轮检测和结束操作必须串行，避免多标签页或倒计时
// 与发送请求并发时出现重复轮次、消息写入已结束会话或反馈记录互相覆盖。
var sessionOperationLocks [sessionOperationLockCount]sync.Mutex

func getSessionOperationLock(sessionID uint) *sync.Mutex {
	return &sessionOperationLocks[sessionID%sessionOperationLockCount]
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	RelationshipType     string `json:"relationship_type" binding:"required"`
	Topic                string `json:"topic" binding:"required"`
	Mode                 string `json:"mode" binding:"required"`
	TargetLanguage       string `json:"target_language" binding:"required"`
	LLMRoleID            string `json:"llm_role_id"`
	LLMCountry           string `json:"llm_country"`
	FeedbackMode         string `json:"feedback_mode" binding:"required"`
	AISuggestionsEnabled *bool  `json:"ai_suggestions_enabled"`
}

// createSession 创建新的对话会话
func createSession(c *gin.Context) {
	userID := getCurrentUserID(c)

	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 校验模式
	if req.Mode != ModeUserL2 && req.Mode != ModeLLML2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的对话模式"})
		return
	}
	if req.FeedbackMode != FeedbackComplete && req.FeedbackMode != FeedbackRounds5 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话模式"})
		return
	}
	req.RelationshipType = strings.TrimSpace(req.RelationshipType)
	req.Topic = strings.TrimSpace(req.Topic)
	req.TargetLanguage = strings.ToUpper(strings.TrimSpace(req.TargetLanguage))
	req.LLMCountry = strings.ToUpper(strings.TrimSpace(req.LLMCountry))
	if req.RelationshipType == "" || utf8.RuneCountInString(req.RelationshipType) > 50 ||
		req.Topic == "" || utf8.RuneCountInString(req.Topic) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "关系或主题为空，或长度超过限制"})
		return
	}
	if !isValidTargetLanguage(req.TargetLanguage) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的目标语言"})
		return
	}
	var sessionUser User
	if err := db.Select("id", "country").First(&sessionUser, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取用户信息失败"})
		return
	}
	if req.LLMCountry != "" {
		if !isPersonaCountryAllowed(req.Mode, req.TargetLanguage, sessionUser.Country, req.LLMCountry) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "对话对象国家与当前模式或目标语言不匹配"})
			return
		}
		// 保留旧字段的默认值以兼容旧版客户端和历史查询；新会话以人物快照为准。
		req.LLMRoleID = defaultLLMRoleID
	} else {
		if strings.TrimSpace(req.LLMRoleID) != "" && !isValidLLMRoleID(req.LLMRoleID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 LLM 角色"})
			return
		}
		req.LLMRoleID = normalizeLLMRoleID(req.LLMRoleID)
	}
	aiSuggestionsEnabled := resolveAISuggestionsEnabled(req.AISuggestionsEnabled)

	var activeCount int64
	if err := db.Model(&ConversationSession{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Count(&activeCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查活跃会话失败"})
		return
	}
	if activeCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "已有未结束的会话，请先继续或结束该会话"})
		return
	}

	// 获取 bot 用户
	var botUser User
	if err := db.Where("role = ?", RoleBot).First(&botUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM Bot 用户不存在"})
		return
	}

	session := ConversationSession{
		UserID:               userID,
		BotUserID:            botUser.ID,
		RelationshipType:     req.RelationshipType,
		Topic:                req.Topic,
		Mode:                 req.Mode,
		TargetLanguage:       req.TargetLanguage,
		LLMRoleID:            req.LLMRoleID,
		FeedbackMode:         req.FeedbackMode,
		AISuggestionsEnabled: aiSuggestionsEnabled,
		RoundCount:           0,
		IsActive:             true,
	}
	if req.LLMCountry != "" {
		persona := generateConversationPersona(
			req.LLMCountry,
			req.Mode,
			req.TargetLanguage,
			req.RelationshipType,
			req.Topic,
		)
		applyPersonaToSession(&session, persona)
		if !hasCompleteGeneratedPersona(session) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "生成对话对象失败"})
			return
		}
	}

	if err := db.Transaction(func(tx *gorm.DB) error {
		// 锁定账号行，使多进程部署下的并发创建也只能产生一个活跃会话。
		var owner User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&owner, userID).Error; err != nil {
			return err
		}
		var activeCount int64
		if err := tx.Model(&ConversationSession{}).
			Where("user_id = ? AND is_active = ?", userID, true).
			Count(&activeCount).Error; err != nil {
			return err
		}
		if activeCount > 0 {
			return errActiveSessionExists
		}
		if err := tx.Create(&session).Error; err != nil {
			return err
		}
		// GORM 会对带 default 标签的 false 零值应用默认值，因此显式写回用户选择。
		if !aiSuggestionsEnabled {
			if err := tx.Model(&session).Update("ai_suggestions_enabled", false).Error; err != nil {
				return err
			}
			session.AISuggestionsEnabled = false
		}
		return nil
	}); err != nil {
		if errors.Is(err, errActiveSessionExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "已有未结束的会话，请先继续或结束该会话"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败"})
		return
	}

	c.JSON(http.StatusOK, session)
}

func isValidTargetLanguage(language string) bool {
	switch strings.ToUpper(strings.TrimSpace(language)) {
	case "EN", "ZH", "JP", "KR", "FR", "DE":
		return true
	default:
		return false
	}
}

func resolveAISuggestionsEnabled(value *bool) bool {
	if value == nil {
		return true
	}
	return *value
}

// getActiveSession 获取当前用户的活跃会话
func getActiveSession(c *gin.Context) {
	userID := getCurrentUserID(c)

	var session ConversationSession
	if err := db.Where("user_id = ? AND is_active = ?", userID, true).
		Order("created_at DESC").First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"session": nil})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取活跃会话失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

// endSession 结束会话，生成汇总反馈
func endSession(c *gin.Context) {
	userID := getCurrentUserID(c)
	sessionID := c.Param("id")

	id64, err := strconv.ParseUint(sessionID, 10, 64)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话ID"})
		return
	}
	id := uint(id64)

	operationLock := getSessionOperationLock(id)
	operationLock.Lock()
	defer operationLock.Unlock()

	var session ConversationSession
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	// 结束前读取用户信息。若此处失败，会话仍保持活跃，前后端状态不会分叉。
	var user User
	if session.AISuggestionsEnabled && session.FeedbackMode == FeedbackComplete {
		if err := db.First(&user, userID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取用户信息失败"})
			return
		}
	}

	// 先通过条件更新抢占“结束会话”操作，防止重复请求多次调用模型；
	// 同时阻止分析期间继续写入新消息。
	claim := db.Model(&ConversationSession{}).
		Where("id = ? AND user_id = ? AND is_active = ?", session.ID, userID, true).
		Update("is_active", false)
	if claim.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "结束会话失败"})
		return
	}
	if claim.RowsAffected == 0 {
		// 另一个请求可能刚完成结束流程，重新读取，避免返回抢占前的旧总结。
		if err := db.First(&session, session.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取会话状态失败"})
			return
		}
		var existingIssueCount int64
		if err := db.Model(&GrammarError{}).
			Where("user_id = ? AND session_id = ? AND error_type <> ?", userID, session.ID, ErrorTypeNoIssue).
			Count(&existingIssueCount).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取会话反馈失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":          "会话已结束",
			"summary_feedback": session.SummaryFeedback,
			"error_count":      existingIssueCount,
			"round_count":      session.RoundCount,
			"already_ended":    true,
		})
		return
	}
	session.IsActive = false

	// 关闭 AI 建议时只结束会话。逐轮反馈模式的结果已在每轮保存，
	// 结束时不再生成整场分析，以免覆盖逐轮记录。
	if !session.AISuggestionsEnabled || session.FeedbackMode != FeedbackComplete {
		var existingIssueCount int64
		if session.AISuggestionsEnabled {
			if err := db.Model(&GrammarError{}).
				Where("user_id = ? AND session_id = ? AND error_type <> ?", userID, session.ID, ErrorTypeNoIssue).
				Count(&existingIssueCount).Error; err != nil {
				c.JSON(http.StatusOK, gin.H{
					"message":          "会话已结束，但暂时无法读取逐轮反馈统计",
					"summary_feedback": "",
					"error_count":      0,
					"round_count":      session.RoundCount,
					"feedback_pending": true,
				})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"message":          "会话已结束",
			"summary_feedback": "",
			"error_count":      existingIssueCount,
			"round_count":      session.RoundCount,
			"analysis_skipped": true,
		})
		return
	}

	// 获取会话所有消息
	var messages []Message
	if err := db.Where("session_id = ?", session.ID).
		Order("created_at ASC").
		Preload("Sender").Preload("Receiver").
		Find(&messages).Error; err != nil {
		// 会话已经成功关闭，不能再返回一个会诱导前端恢复输入框的失败状态。
		// 数据库恢复后仍可从历史消息重新生成反馈。
		c.JSON(http.StatusOK, gin.H{
			"message":          "会话已结束，但暂时无法读取会话反馈",
			"summary_feedback": buildFallbackSessionSummary(session, user),
			"error_count":      0,
			"round_count":      session.RoundCount,
			"feedback_pending": true,
		})
		return
	}

	// 生成汇总反馈
	summaryFeedback := ""
	sessionErrorCount := 0
	if len(messages) >= 2 {
		summaryFeedback, sessionErrorCount = generateAndStoreSessionFeedback(session, messages, user)
	}

	// 更新会话状态
	if err := db.Model(&session).Updates(map[string]interface{}{
		"summary_feedback": summaryFeedback,
	}).Error; err != nil {
		// 结束状态已持久化，仍把刚生成的总结返回给当前请求，避免前端误判会话仍活跃。
		c.JSON(http.StatusOK, gin.H{
			"message":          "会话已结束，但反馈暂未保存",
			"summary_feedback": summaryFeedback,
			"error_count":      sessionErrorCount,
			"round_count":      session.RoundCount,
			"feedback_pending": true,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "会话已结束",
		"summary_feedback": summaryFeedback,
		"error_count":      sessionErrorCount,
		"round_count":      session.RoundCount,
	})
}

// getSessionMessages 获取会话消息
func getSessionMessages(c *gin.Context) {
	userID := getCurrentUserID(c)
	sessionID := c.Param("id")

	id, err := strconv.Atoi(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话ID"})
		return
	}

	var session ConversationSession
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	var messages []Message
	if err := db.Where("session_id = ?", session.ID).
		Order("created_at ASC").
		Preload("Sender").Preload("Receiver").
		Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取会话消息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session":  session,
		"messages": messages,
	})
}

// getSessionFeedback 获取会话汇总反馈
func getSessionFeedback(c *gin.Context) {
	userID := getCurrentUserID(c)
	sessionID := c.Param("id")

	id, err := strconv.Atoi(sessionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的会话ID"})
		return
	}

	var session ConversationSession
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"summary_feedback": session.SummaryFeedback,
		"round_count":      session.RoundCount,
	})
}

// getLLMBotInfo 获取 LLM Bot 用户信息
func getLLMBotInfo(c *gin.Context) {
	var botUser User
	if err := db.Where("role = ?", RoleBot).First(&botUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Bot 用户不存在"})
		return
	}
	c.JSON(http.StatusOK, botUser)
}
