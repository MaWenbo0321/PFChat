package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// 实时检测接口 (Grammarly 风格, 前端每5秒调用一次)
// ============================================================================

// RealtimeCheckRequest 实时检测请求
type RealtimeCheckRequest struct {
	ReceiverID uint   `json:"receiver_id" binding:"required"`
	Content    string `json:"content" binding:"required"`
}

// RealtimeCheckError 单个错误信息 (带位置)
type RealtimeCheckError struct {
	StartIndex    int    `json:"start_index"`
	EndIndex      int    `json:"end_index"`
	OriginalText  string `json:"original_text"`
	Suggestion    string `json:"suggestion"`
	Explanation   string `json:"explanation"`
	ErrorType     string `json:"error_type"`
	ErrorRecordID uint   `json:"error_record_id"`
}

// RealtimeCheckResponse 实时检测响应
type RealtimeCheckResponse struct {
	HasError bool                 `json:"has_error"`
	Errors   []RealtimeCheckError `json:"errors"`
}

func realtimeCheck(c *gin.Context) {
	userID := getCurrentUserID(c)

	var req RealtimeCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 检查是否有最近的缓存结果 (30秒内相同内容)
	var recentError GrammarError
	thirtySecondsAgo := time.Now().Add(-30 * time.Second)
	err := db.Where(
		"user_id = ? AND original_text = ? AND created_at > ?",
		userID, req.Content, thirtySecondsAgo,
	).Order("created_at DESC").First(&recentError).Error

	if err == nil && recentError.LLMSuggestion != "" {
		// 有缓存, 直接返回
		log.Printf("⚡ 实时检测: 重用缓存结果, 记录ID: %d", recentError.ID)
		startIdx, endIdx := findErrorPosition(req.Content, recentError.OriginalText)
		c.JSON(http.StatusOK, RealtimeCheckResponse{
			HasError: true,
			Errors: []RealtimeCheckError{
				{
					StartIndex:    startIdx,
					EndIndex:      endIdx,
					OriginalText:  recentError.OriginalText,
					Suggestion:    recentError.LLMSuggestion,
					Explanation:   recentError.LLMExplanation,
					ErrorType:     recentError.ErrorType,
					ErrorRecordID: recentError.ID,
				},
			},
		})
		return
	}

	// 获取发送者和接收者信息
	var sender, receiver User
	if err := db.First(&sender, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
		return
	}
	if err := db.First(&receiver, req.ReceiverID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "接收者不存在"})
		return
	}

	// 获取历史消息
	var historyMessages []Message
	db.Where(
		"((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?))",
		userID, req.ReceiverID, req.ReceiverID, userID,
	).Order("created_at DESC").Limit(10).Preload("Sender").Preload("Receiver").Find(&historyMessages)

	tempMessage := Message{
		SenderID:   userID,
		ReceiverID: req.ReceiverID,
		Content:    req.Content,
	}

	// 调用 LLM 检测
	prompt := buildCombinedPrompt(historyMessages, tempMessage, sender, receiver)
	result, err := callDashScopeAPI(prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM检查失败: " + err.Error()})
		return
	}

	if !result.HasError {
		c.JSON(http.StatusOK, RealtimeCheckResponse{
			HasError: false,
			Errors:   []RealtimeCheckError{},
		})
		return
	}

	errorType := result.ErrorType
	if errorType != "语言语用失误" && errorType != "社会语用失误" {
		errorType = "语言语用失误"
	}

	// 保存到数据库
	grammarError := GrammarError{
		UserID:         userID,
		MessageID:      0,
		OriginalText:   req.Content,
		LLMSuggestion:  result.Suggestion,
		LLMExplanation: result.Explanation,
		ErrorType:      errorType,
	}

	var errorRecordID uint = 0
	if err := db.Create(&grammarError).Error; err != nil {
		log.Printf("保存实时检测错误记录失败: %v", err)
	} else {
		errorRecordID = grammarError.ID
	}

	// 计算错误位置 (整个文本)
	startIdx, endIdx := 0, utf8.RuneCountInString(req.Content)

	c.JSON(http.StatusOK, RealtimeCheckResponse{
		HasError: true,
		Errors: []RealtimeCheckError{
			{
				StartIndex:    startIdx,
				EndIndex:      endIdx,
				OriginalText:  req.Content,
				Suggestion:    result.Suggestion,
				Explanation:   result.Explanation,
				ErrorType:     errorType,
				ErrorRecordID: errorRecordID,
			},
		},
	})
}

// findErrorPosition 查找错误文本在完整内容中的位置
func findErrorPosition(fullContent, errorText string) (int, int) {
	// 将字符串转为 rune 切片以正确处理中文
	fullRunes := []rune(fullContent)
	errorRunes := []rune(errorText)

	if len(errorRunes) == 0 || len(fullRunes) == 0 {
		return 0, len(fullRunes)
	}

	// 尝试查找子串
	fullStr := string(fullRunes)
	errorStr := string(errorRunes)
	idx := strings.Index(fullStr, errorStr)

	if idx >= 0 {
		startRune := utf8.RuneCountInString(fullStr[:idx])
		endRune := startRune + len(errorRunes)
		return startRune, endRune
	}

	// 找不到则标记整个文本
	return 0, len(fullRunes)
}

// ============================================================================
// 获取消息关联的语用错误 (接收方查看)
// ============================================================================

// GetMessageErrorsRequest 请求
type GetMessageErrorsRequest struct {
	MessageIDs []uint `json:"message_ids" binding:"required"`
}

// MessageErrorInfo 响应中的错误信息
type MessageErrorInfo struct {
	MessageID   uint   `json:"message_id"`
	ErrorType   string `json:"error_type"`
	Suggestion  string `json:"suggestion"`
	Explanation string `json:"explanation"`
}

func getMessageErrorsByIds(c *gin.Context) {
	var req GetMessageErrorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	if len(req.MessageIDs) == 0 {
		c.JSON(http.StatusOK, []MessageErrorInfo{})
		return
	}

	// 查询这些消息关联的错误记录
	var errors []GrammarError
	db.Where("message_id IN ? AND message_id > 0", req.MessageIDs).Find(&errors)

	result := make([]MessageErrorInfo, 0, len(errors))
	for _, err := range errors {
		result = append(result, MessageErrorInfo{
			MessageID:   err.MessageID,
			ErrorType:   err.ErrorType,
			Suggestion:  err.LLMSuggestion,
			Explanation: err.LLMExplanation,
		})
	}

	c.JSON(http.StatusOK, result)
}

// ============================================================================
// AI Chat 接口
// ============================================================================

// AIChatRequest AI对话请求
type AIChatRequest struct {
	Message    string `json:"message" binding:"required"`
	ChatUserID uint   `json:"chat_user_id"` // 当前聊天对象ID, 用于提供上下文
}

// AIChatResponse AI对话响应
type AIChatResponse struct {
	Reply string `json:"reply"`
}

func aiChat(c *gin.Context) {
	userID := getCurrentUserID(c)

	var req AIChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 获取用户信息
	var user User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
		return
	}

	// 获取聊天对象信息 (如果有)
	var chatUserInfo string
	if req.ChatUserID > 0 {
		var chatUser User
		if err := db.First(&chatUser, req.ChatUserID).Error; err == nil {
			chatUserInfo = fmt.Sprintf("当前用户正在与来自%s的%s聊天。", getCountryName(chatUser.Country), chatUser.Username)
		}
	}

	// 获取最近的 AI 对话历史 (最近5轮)
	var history []AIChatHistory
	db.Where("user_id = ?", userID).Order("created_at DESC").Limit(10).Find(&history)

	// 构建 AI prompt
	prompt := buildAIChatPrompt(user, chatUserInfo, history, req.Message)

	// 调用 LLM
	reply, err := callAIChatAPI(prompt)
	if err != nil {
		log.Printf("AI Chat API error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI 回复失败"})
		return
	}

	// 保存对话历史
	db.Create(&AIChatHistory{
		UserID:     userID,
		ChatUserID: req.ChatUserID,
		Role:       "user",
		Content:    req.Message,
	})
	db.Create(&AIChatHistory{
		UserID:     userID,
		ChatUserID: req.ChatUserID,
		Role:       "assistant",
		Content:    reply,
	})

	c.JSON(http.StatusOK, AIChatResponse{Reply: reply})
}

// buildAIChatPrompt 构建 AI Chat 的 prompt
func buildAIChatPrompt(user User, chatUserInfo string, history []AIChatHistory, currentMessage string) string {
	var sb strings.Builder

	sb.WriteString("你是一个跨文化沟通助手，专门帮助用户理解不同文化之间的语用差异和沟通技巧。\n")
	sb.WriteString("你精通Thomas的语用失误理论，了解语言语用失误(Pragmalinguistic failure)和社会语用失误(Sociopragmatic failure)的区别。\n")
	sb.WriteString("请用友好、专业的方式回答用户的问题。\n\n")

	sb.WriteString(fmt.Sprintf("用户信息: %s, 来自%s\n", user.Username, getCountryName(user.Country)))
	if chatUserInfo != "" {
		sb.WriteString(chatUserInfo + "\n")
	}
	sb.WriteString("\n")

	// 添加对话历史 (倒序变正序)
	if len(history) > 0 {
		sb.WriteString("最近对话:\n")
		for i := len(history) - 1; i >= 0; i-- {
			h := history[i]
			if h.Role == "user" {
				sb.WriteString(fmt.Sprintf("用户: %s\n", h.Content))
			} else {
				sb.WriteString(fmt.Sprintf("助手: %s\n", h.Content))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("当前问题: %s\n", currentMessage))
	sb.WriteString("\n请根据用户的文化背景选择回复语言。如果用户来自中国，用中文回复；否则用英文回复。")

	return sb.String()
}

// callAIChatAPI 调用 DashScope API 进行 AI 对话 (返回纯文本)
func callAIChatAPI(prompt string) (string, error) {
	apiKey := "sk-8ab77da79b894ba6beb61c9190c74602"
	if apiKey == "" {
		return "", fmt.Errorf("DASHSCOPE_API_KEY 未设置")
	}

	reqBody := DashScopeRequest{
		Model: defaultModel,
		Input: DashScopeInput{
			Messages: []DashScopeMessage{
				{
					Role:    "user",
					Content: prompt,
				},
			},
		},
		Parameters: DashScopeParameters{
			ResultFormat: "message",
			Temperature:  0.7,
			MaxTokens:    1024,
		},
	}

	respBody, err := makeDashScopeRequest(apiKey, reqBody)
	if err != nil {
		return "", err
	}

	// 提取纯文本回复
	if len(respBody.Output.Choices) > 0 {
		return respBody.Output.Choices[0].Message.Content, nil
	}
	if respBody.Output.Text != "" {
		return respBody.Output.Text, nil
	}

	return "抱歉，我暂时无法回答。", nil
}
