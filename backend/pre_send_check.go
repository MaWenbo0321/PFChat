package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// CheckMessageRequest 发送前检查请求
type CheckMessageRequest struct {
	ReceiverID uint   `json:"receiver_id" binding:"required"`
	Content    string `json:"content" binding:"required"`
}

// CheckMessageResponse 发送前检查响应
type CheckMessageResponse struct {
	HasError      bool   `json:"has_error"`
	Suggestion    string `json:"suggestion"`
	Explanation   string `json:"explanation"`
	ErrorType     string `json:"error_type"`
	ErrorRecordID uint   `json:"error_record_id"`
}

// 发送前检查消息是否有语用失误
func checkMessageBeforeSend(c *gin.Context) {
	userID := getCurrentUserID(c)

	var req CheckMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 检查是否有最近相同内容的检测结果（防止重复检测）
	// 只查找 message_id = 0 的记录（未发送的预检记录）
	var recentError GrammarError
	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	err := db.Where(
		"user_id = ? AND original_text = ? AND message_id = 0 AND created_at > ?",
		userID, req.Content, oneMinuteAgo,
	).Order("created_at DESC").First(&recentError).Error

	if err == nil {
		// 找到了最近的相同检查结果，直接返回
		log.Printf("⚡ 重用最近的检查结果，记录ID: %d", recentError.ID)
		c.JSON(http.StatusOK, CheckMessageResponse{
			HasError:      recentError.LLMSuggestion != "",
			Suggestion:    recentError.LLMSuggestion,
			Explanation:   recentError.LLMExplanation,
			ErrorType:     recentError.ErrorType,
			ErrorRecordID: recentError.ID,
		})
		return
	}

	var sender, receiver User
	if err := db.First(&sender, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户信息失败"})
		return
	}
	if err := db.First(&receiver, req.ReceiverID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "接收者不存在"})
		return
	}

	// 只获取已发送的消息作为历史记录
	// 注意：这里从 Message 表查询，而不是从 GrammarError 表
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

	// 使用合并的 prompt
	prompt := buildCombinedPrompt(historyMessages, tempMessage, sender, receiver)
	result, err := callDashScopeAPI(prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM检查失败: " + err.Error()})
		return
	}

	errorType := result.ErrorType
	if errorType != "语言语用失误" && errorType != "社会语用失误" {
		errorType = "语言语用失误"
	}

	var errorRecordID uint = 0

	// 只有检测到错误时才创建预检记录
	// message_id = 0 表示这是一个预检记录，尚未关联到实际发送的消息
	if result.HasError {
		grammarError := GrammarError{
			UserID:         userID,
			MessageID:      0, // 0 表示预检记录，尚未发送
			OriginalText:   req.Content,
			LLMSuggestion:  result.Suggestion,
			LLMExplanation: result.Explanation,
			ErrorType:      errorType,
		}

		if err := db.Create(&grammarError).Error; err != nil {
			log.Printf("保存语用错误预检记录失败: %v", err)
		} else {
			errorRecordID = grammarError.ID
			log.Printf("语用错误预检记录已保存，记录ID: %d (message_id=0，待发送)", errorRecordID)
		}
	}

	c.JSON(http.StatusOK, CheckMessageResponse{
		HasError:      result.HasError,
		Suggestion:    result.Suggestion,
		Explanation:   result.Explanation,
		ErrorType:     errorType,
		ErrorRecordID: errorRecordID,
	})
}

// 定期清理未发送的预检记录（可选，防止数据库膨胀）
func cleanupUnsentPreCheckRecords() {
	// 删除超过24小时且 message_id = 0 的预检记录
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	result := db.Where("message_id = 0 AND created_at < ?", oneDayAgo).Delete(&GrammarError{})
	if result.Error != nil {
		log.Printf("清理未发送预检记录失败: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("已清理 %d 条未发送的预检记录", result.RowsAffected)
	}
}