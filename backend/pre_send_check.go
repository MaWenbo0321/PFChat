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
    
    var recentError GrammarError
    oneMinuteAgo := time.Now().Add(-1 * time.Minute)
    err := db.Where(
        "user_id = ? AND original_text = ? AND created_at > ?",
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

	// 🔧 使用合并的 prompt
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
			log.Printf("保存语用错误记录失败: %v", err)
		} else {
			errorRecordID = grammarError.ID
			log.Printf("语用错误已保存到数据库，记录ID: %d", errorRecordID)
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