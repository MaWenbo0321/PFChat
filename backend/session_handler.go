package main

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	RelationshipType string `json:"relationship_type" binding:"required"`
	Topic            string `json:"topic" binding:"required"`
	Mode             string `json:"mode" binding:"required"`
	TargetLanguage   string `json:"target_language" binding:"required"`
	FeedbackMode     string `json:"feedback_mode" binding:"required"`
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

	// 获取 bot 用户
	var botUser User
	if err := db.Where("role = ?", RoleBot).First(&botUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM Bot 用户不存在"})
		return
	}

	// 将当前用户的所有活跃会话设为非活跃
	db.Model(&ConversationSession{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Update("is_active", false)

	session := ConversationSession{
		UserID:           userID,
		BotUserID:        botUser.ID,
		RelationshipType: req.RelationshipType,
		Topic:            req.Topic,
		Mode:             req.Mode,
		TargetLanguage:   req.TargetLanguage,
		FeedbackMode:     req.FeedbackMode,
		RoundCount:       0,
		IsActive:         true,
	}

	if err := db.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// getActiveSession 获取当前用户的活跃会话
func getActiveSession(c *gin.Context) {
	userID := getCurrentUserID(c)

	var session ConversationSession
	if err := db.Where("user_id = ? AND is_active = ?", userID, true).
		Order("created_at DESC").First(&session).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"session": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session": session})
}

// endSession 结束会话，生成汇总反馈
func endSession(c *gin.Context) {
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

	// 获取会话所有消息
	var messages []Message
	db.Where("session_id = ?", session.ID).
		Order("created_at ASC").
		Preload("Sender").Preload("Receiver").
		Find(&messages)

	// 获取当前用户
	var user User
	db.First(&user, userID)

	// 生成汇总反馈
	summaryFeedback := ""
	if len(messages) >= 2 {
		summaryFeedback = generateSessionSummary(session, messages, user)
	}

	// 更新会话状态
	db.Model(&session).Updates(map[string]interface{}{
		"is_active":        false,
		"summary_feedback": summaryFeedback,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":          "会话已结束",
		"summary_feedback": summaryFeedback,
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
	db.Where("session_id = ?", session.ID).
		Order("created_at ASC").
		Preload("Sender").Preload("Receiver").
		Find(&messages)

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
