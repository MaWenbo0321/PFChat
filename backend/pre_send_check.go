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
	HasError          bool   `json:"has_error"`
	Suggestion        string `json:"suggestion"`
	Explanation       string `json:"explanation"`
	ErrorType         string `json:"error_type"`
	ErrorRecordID     uint   `json:"error_record_id"`
	OverallEvaluation string `json:"overall_evaluation"` // "good" / "improvable" / "problematic"
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
	var recentError GrammarError
	oneMinuteAgo := time.Now().Add(-1 * time.Minute)
	err := db.Where(
		"user_id = ? AND original_text = ? AND message_id = 0 AND created_at > ?",
		userID, req.Content, oneMinuteAgo,
	).Order("created_at DESC").First(&recentError).Error

	if err == nil {
		log.Printf("⚡ 重用最近的检查结果，记录ID: %d", recentError.ID)
		overallEvaluation := "good"
		hasError := recentError.LLMSuggestion != ""
		if hasError {
			if IsProblematicErrorType(recentError.ErrorType) {
				overallEvaluation = "problematic"
			} else {
				overallEvaluation = "improvable"
			}
		}
		c.JSON(http.StatusOK, CheckMessageResponse{
			HasError:          hasError,
			Suggestion:        recentError.LLMSuggestion,
			Explanation:       recentError.LLMExplanation,
			ErrorType:         recentError.ErrorType,
			ErrorRecordID:     recentError.ID,
			OverallEvaluation: overallEvaluation,
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

	prompt := buildCombinedPrompt(historyMessages, tempMessage, sender, receiver)
	result, err := callDashScopeAPI(prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "LLM检查失败: " + err.Error()})
		return
	}

	errorType := result.ErrorType
	if !IsValidErrorType(errorType) {
		errorType = ErrorTypePragmalinguistic
	}

	overallEvaluation := result.OverallEvaluation
	if overallEvaluation == "" {
		overallEvaluation = "good"
		if result.HasError {
			if IsProblematicErrorType(errorType) {
				overallEvaluation = "problematic"
			} else {
				overallEvaluation = "improvable"
			}
		}
	}

	var errorRecordID uint = 0

	if result.HasError {
		grammarError := GrammarError{
			UserID:         userID,
			MessageID:      0,
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
		HasError:          result.HasError,
		Suggestion:        result.Suggestion,
		Explanation:       result.Explanation,
		ErrorType:         errorType,
		ErrorRecordID:     errorRecordID,
		OverallEvaluation: overallEvaluation,
	})
}

// 定期清理未发送的预检记录
func cleanupUnsentPreCheckRecords() {
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	result := db.Where("message_id = 0 AND created_at < ?", oneDayAgo).Delete(&GrammarError{})
	if result.Error != nil {
		log.Printf("清理未发送预检记录失败: %v", result.Error)
	} else if result.RowsAffected > 0 {
		log.Printf("已清理 %d 条未发送的预检记录", result.RowsAffected)
	}
}
