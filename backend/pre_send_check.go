package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CheckMessageRequest 发送前检查请求
type CheckMessageRequest struct {
	ReceiverID uint   `json:"receiver_id" binding:"required"`
	Content    string `json:"content" binding:"required"`
}

// CheckMessageResponse 发送前检查响应
type CheckMessageResponse struct {
	HasError    bool   `json:"has_error"`
	Suggestion  string `json:"suggestion"`
	Explanation string `json:"explanation"`
	ErrorType   string `json:"error_type"` // "语言语用失误" 或 "社会语用失误"
}

// 发送前检查消息是否有语用失误
func checkMessageBeforeSend(c *gin.Context) {
	userID := getCurrentUserID(c)

	var req CheckMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
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

	// 获取最近的历史消息作为上下文
	var historyMessages []Message
	db.Where(
		"((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?))",
		userID, req.ReceiverID, req.ReceiverID, userID,
	).Order("created_at DESC").Limit(10).Preload("Sender").Preload("Receiver").Find(&historyMessages)

	// 创建临时消息对象用于检查
	tempMessage := Message{
		SenderID:   userID,
		ReceiverID: req.ReceiverID,
		Content:    req.Content,
	}

	// 构建提示词
	prompt := buildPrompt(historyMessages, tempMessage, sender, receiver)

	// 调用 LLM API
	result, err := callOllamaAPI(prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM检查失败: " + err.Error()})
		return
	}

	// 确保错误类型有效
	errorType := result.ErrorType
	if errorType != "语言语用失误" && errorType != "社会语用失误" {
		errorType = "语言语用失误" // 默认值
	}

	// 返回检查结果
	c.JSON(http.StatusOK, CheckMessageResponse{
		HasError:    result.HasError,
		Suggestion:  result.Suggestion,
		Explanation: result.Explanation,
		ErrorType:   errorType,
	})
}
