package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// 用户相关处理函数
// ============================================================================

// 获取用户列表（排除当前用户）
func getUsers(c *gin.Context) {
	currentUserID := getCurrentUserID(c)

	var users []User
	if err := db.Where("id != ?", currentUserID).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// 获取用户信息
func getUserInfo(c *gin.Context) {
	userID := c.Param("id")

	var user User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// ============================================================================
// 消息相关处理函数
// ============================================================================

// 获取与某个用户的聊天记录
func getMessages(c *gin.Context) {
	currentUserID := getCurrentUserID(c)
	otherUserID := c.Param("userId")

	var messages []Message
	err := db.Where(
		"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		currentUserID, otherUserID, otherUserID, currentUserID,
	).Order("sent_at ASC").Preload("Sender").Preload("Receiver").Find(&messages).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息失败"})
		return
	}

	// 标记消息为已读
	db.Model(&Message{}).Where(
		"sender_id = ? AND receiver_id = ? AND is_read = ?",
		otherUserID, currentUserID, false,
	).Update("is_read", true)

	c.JSON(http.StatusOK, messages)
}

// HTTP 发送消息接口
func sendMessage(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	senderID := getCurrentUserID(c)

	// 检查接收者是否存在
	var receiver User
	if err := db.First(&receiver, req.ReceiverID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "接收者不存在"})
		return
	}

	// 保存消息
	message := Message{
		SenderID:   senderID,
		ReceiverID: req.ReceiverID,
		Content:    req.Content,
	}

	if err := db.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送消息失败"})
		return
	}

	// 预加载关联数据
	db.Preload("Sender").Preload("Receiver").First(&message, message.ID)

	// 🔧 注释掉或删除异步语法检查 - 因为已在发送前检查过
	// go checkGrammar(senderID, message)

	// 通过 WebSocket 发送给接收者
	wsMsg := WSMessage{
		Type:      "message",
		Data:      message,
		Timestamp: message.CreatedAt,
	}

	msgBytes, _ := json.Marshal(wsMsg)
	if globalHub != nil {
		globalHub.sendToUser(req.ReceiverID, msgBytes)
		// 同时发送给发送者（用于多设备同步）
		globalHub.sendToUser(senderID, msgBytes)
	}

	c.JSON(http.StatusOK, message)
}

// WebSocket 消息处理
func handleChatMessage(client *Client, data interface{}) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	receiverID := uint(dataMap["receiver_id"].(float64))
	content := dataMap["content"].(string)

	// 保存消息
	message := Message{
		SenderID:   client.userID,
		ReceiverID: receiverID,
		Content:    content,
		CreatedAt:  time.Now(),
	}

	if err := db.Create(&message).Error; err != nil {
		log.Println("Save message error:", err)
		return
	}

	// 预加载关联数据
	db.Preload("Sender").Preload("Receiver").First(&message, message.ID)

	// 发送给接收者
	wsMsg := WSMessage{
		Type:      "message",
		Data:      message,
		Timestamp: time.Now(),
	}

	msgBytes, _ := json.Marshal(wsMsg)
	client.hub.sendToUser(receiverID, msgBytes)

	// 异步语法检查
	go checkGrammar(client.userID, message)
}

// 删除单条消息
func deleteMessage(c *gin.Context) {
	messageID := c.Param("id")
	currentUserID := getCurrentUserID(c)

	id, err := strconv.Atoi(messageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	// 查询消息
	var message Message
	if err := db.First(&message, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "消息不存在"})
		return
	}

	// 只能删除自己发送的消息，或者管理员可以删除任何消息
	if message.SenderID != currentUserID && !isCurrentUserAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除此消息"})
		return
	}

	// 删除消息
	if err := db.Delete(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功", "message_id": id})
}

// 清空与某个用户的聊天记录
func clearChatHistory(c *gin.Context) {
	otherUserID := c.Param("userId")
	currentUserID := getCurrentUserID(c)

	otherID, err := strconv.Atoi(otherUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 删除双方的所有消息
	result := db.Where(
		"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		currentUserID, otherID, otherID, currentUserID,
	).Delete(&Message{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "清空失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "聊天记录已清空",
		"deleted_count": result.RowsAffected,
	})
}

// 删除自己发送的所有消息（针对特定对话）
func deleteMyMessages(c *gin.Context) {
	otherUserID := c.Param("userId")
	currentUserID := getCurrentUserID(c)

	otherID, err := strconv.Atoi(otherUserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 只删除自己发送的消息
	result := db.Where(
		"sender_id = ? AND receiver_id = ?",
		currentUserID, otherID,
	).Delete(&Message{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "已删除我发送的消息",
		"deleted_count": result.RowsAffected,
	})
}

// ============================================================================
// 语法错误相关处理函数（已扩展支持错误类型）
// ============================================================================

// GetGrammarErrorsRequest 获取语法错误请求
type GetGrammarErrorsRequest struct {
	ErrorType string `form:"error_type"` // 错误类型筛选: "错误1", "错误2", "all"
}

// GetGrammarErrorsResponse 获取语法错误响应
type GetGrammarErrorsResponse struct {
	Errors     []GrammarError         `json:"errors"`
	Statistics GrammarErrorStatistics `json:"statistics"`
}

// GrammarErrorStatistics 语法错误统计
type GrammarErrorStatistics struct {
	Total      int64          `json:"total"`
	ByType     map[string]int `json:"by_type"`
	TodayCount int64          `json:"today_count"`
	WeekCount  int64          `json:"week_count"`
}

// 获取当前用户的语法错误记录（支持按类型筛选）
func getGrammarErrors(c *gin.Context) {
	userID := getCurrentUserID(c)

	var req GetGrammarErrorsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 构建查询
	query := db.Where("user_id = ?", userID)

	// 按错误类型筛选
	if req.ErrorType != "" && req.ErrorType != "all" {
		query = query.Where("error_type = ?", req.ErrorType)
	}

	// 获取错误记录
	var errors []GrammarError
	if err := query.Order("created_at DESC").Find(&errors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取错误记录失败"})
		return
	}

	// 获取统计信息
	statistics := getGrammarErrorStatistics(userID)

	c.JSON(http.StatusOK, GetGrammarErrorsResponse{
		Errors:     errors,
		Statistics: statistics,
	})
}

// 获取语法错误统计信息
func getGrammarErrorStatistics(userID uint) GrammarErrorStatistics {
	var statistics GrammarErrorStatistics

	// 总数
	db.Model(&GrammarError{}).Where("user_id = ?", userID).Count(&statistics.Total)

	// 按类型统计
	statistics.ByType = make(map[string]int)
	var typeStats []struct {
		ErrorType string
		Count     int
	}
	db.Model(&GrammarError{}).
		Select("error_type, COUNT(*) as count").
		Where("user_id = ?", userID).
		Group("error_type").
		Scan(&typeStats)

	for _, stat := range typeStats {
		statistics.ByType[stat.ErrorType] = stat.Count
	}

	// 今日错误数
	db.Model(&GrammarError{}).
		Where("user_id = ? AND DATE(created_at) = CURDATE()", userID).
		Count(&statistics.TodayCount)

	// 本周错误数
	db.Model(&GrammarError{}).
		Where("user_id = ? AND YEARWEEK(created_at, 1) = YEARWEEK(CURDATE(), 1)", userID).
		Count(&statistics.WeekCount)

	return statistics
}

// 删除语法错误记录
func deleteGrammarError(c *gin.Context) {
	userID := getCurrentUserID(c)
	errorID := c.Param("id")

	// 检查记录是否属于当前用户
	var grammarError GrammarError
	if err := db.Where("id = ? AND user_id = ?", errorID, userID).First(&grammarError).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}

	// 删除记录
	if err := db.Delete(&grammarError).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// 批量删除语法错误记录
func batchDeleteGrammarErrors(c *gin.Context) {
	userID := getCurrentUserID(c)

	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 只删除属于当前用户的记录
	result := db.Where("user_id = ? AND id IN ?", userID, req.IDs).Delete(&GrammarError{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "批量删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "批量删除成功",
		"deleted_count": result.RowsAffected,
	})
}

// 按类型清空语法错误记录
func clearGrammarErrorsByType(c *gin.Context) {
	userID := getCurrentUserID(c)
	errorType := c.Param("type")

	if errorType == "" || errorType == "all" {
		// 清空所有记录
		result := db.Where("user_id = ?", userID).Delete(&GrammarError{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "清空失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":       "清空成功",
			"deleted_count": result.RowsAffected,
		})
	} else {
		// 清空指定类型的记录
		result := db.Where("user_id = ? AND error_type = ?", userID, errorType).Delete(&GrammarError{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "清空失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":       "清空成功",
			"deleted_count": result.RowsAffected,
		})
	}
}

// 更新语法错误的类型
func updateGrammarErrorType(c *gin.Context) {
	userID := getCurrentUserID(c)
	errorID := c.Param("id")

	var req struct {
		ErrorType string `json:"error_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 🔧 验证错误类型 - 使用新的常量
	if req.ErrorType != ErrorTypePragmalinguistic && req.ErrorType != ErrorTypeSociopragmatic {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的错误类型"})
		return
	}

	// 检查记录是否属于当前用户
	var grammarError GrammarError
	if err := db.Where("id = ? AND user_id = ?", errorID, userID).First(&grammarError).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}

	// 更新错误类型
	if err := db.Model(&grammarError).Update("error_type", req.ErrorType).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// ============================================================================
// 管理员专用处理函数
// ============================================================================

// 获取所有用户（管理员专用）
func getAllUsersForAdmin(c *gin.Context) {
	var users []User
	if err := db.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": len(users),
	})
}

// 删除用户（管理员专用）
func deleteUser(c *gin.Context) {
	userID := c.Param("id")
	currentUserID := getCurrentUserID(c)

	id, err := strconv.Atoi(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 不能删除自己
	if uint(id) == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己"})
		return
	}

	// 查询用户
	var user User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 使用事务删除用户及相关数据
	tx := db.Begin()

	// 删除用户发送和接收的所有消息
	if err := tx.Where("sender_id = ? OR receiver_id = ?", id, id).Delete(&Message{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户消息失败"})
		return
	}

	// 删除用户相关的语法错误记录
	if err := tx.Where("user_id = ?", id).Delete(&GrammarError{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户语法记录失败"})
		return
	}

	// 删除用户
	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败"})
		return
	}

	// 提交事务
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "用户删除成功",
		"deleted_user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

// 修改用户角色（管理员专用）
func updateUserRole(c *gin.Context) {
	userID := c.Param("id")
	currentUserID := getCurrentUserID(c)

	id, err := strconv.Atoi(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 不能修改自己的角色
	if uint(id) == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能修改自己的角色"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证角色值
	if req.Role != RoleUser && req.Role != RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色"})
		return
	}

	// 查询用户
	var user User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 更新角色
	if err := db.Model(&user).Update("role", req.Role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户角色失败"})
		return
	}

	user.Role = req.Role

	c.JSON(http.StatusOK, gin.H{
		"message": "用户角色更新成功",
		"user":    user,
	})
}

// 获取用户统计信息（管理员专用）
func getUserStats(c *gin.Context) {
	var stats struct {
		TotalUsers         int64 `json:"total_users"`
		AdminUsers         int64 `json:"admin_users"`
		RegularUsers       int64 `json:"regular_users"`
		TotalMessages      int64 `json:"total_messages"`
		TotalGrammarErrors int64 `json:"total_grammar_errors"`
	}

	// 统计用户总数
	db.Model(&User{}).Count(&stats.TotalUsers)

	// 统计管理员数量
	db.Model(&User{}).Where("role = ?", RoleAdmin).Count(&stats.AdminUsers)

	// 统计普通用户数量
	db.Model(&User{}).Where("role = ?", RoleUser).Count(&stats.RegularUsers)

	// 统计消息总数
	db.Model(&Message{}).Count(&stats.TotalMessages)

	// 统计语法错误总数
	db.Model(&GrammarError{}).Count(&stats.TotalGrammarErrors)

	c.JSON(http.StatusOK, stats)
}
