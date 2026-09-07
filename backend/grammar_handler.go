package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// 获取消息关联的语用错误 (接收方查看)
// ============================================================================

type GetMessageErrorsRequest struct {
	MessageIDs []uint `json:"message_ids" binding:"required"`
}

type MessageErrorInfo struct {
	MessageID       uint   `json:"message_id"`
	ErrorType       string `json:"error_type"`
	IntendedMeaning string `json:"intended_meaning"`
	Suggestion      string `json:"suggestion"`
	Explanation     string `json:"explanation"`
}

func getMessageErrorsByIds(c *gin.Context) {
	userID := getCurrentUserID(c)
	var req GetMessageErrorsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	if len(req.MessageIDs) == 0 {
		c.JSON(http.StatusOK, []MessageErrorInfo{})
		return
	}

	var errors []GrammarError
	if err := db.Where("user_id = ? AND message_id IN ? AND message_id > 0", userID, req.MessageIDs).
		Find(&errors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息反馈失败"})
		return
	}

	result := make([]MessageErrorInfo, 0, len(errors))
	for _, err := range errors {
		result = append(result, MessageErrorInfo{
			MessageID:       err.MessageID,
			ErrorType:       err.ErrorType,
			IntendedMeaning: err.LLMIntendedMeaning,
			Suggestion:      err.LLMSuggestion,
			Explanation:     err.LLMExplanation,
		})
	}

	c.JSON(http.StatusOK, result)
}
