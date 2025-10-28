package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 获取用户列表（管理员查看所有用户，普通用户只看其他用户）
func getUsers(c *gin.Context) {
	currentUserID := getCurrentUserID(c)
	isAdmin := isCurrentUserAdmin(c)

	var users []User
	var query = db

	// 如果不是管理员，排除自己和管理员
	if !isAdmin {
		query = query.Where("id != ? AND role != ?", currentUserID, RoleAdmin)
	}

	if err := query.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	c.JSON(http.StatusOK, users)
}

// 管理员获取所有用户列表（包含详细信息）
func getAllUsersForAdmin(c *gin.Context) {
	var users []User
	var total int64

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	search := c.Query("search")

	query := db.Model(&User{})

	// 搜索功能
	if search != "" {
		query = query.Where("username LIKE ? OR country LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// 获取总数
	query.Count(&total)

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":     users,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己的账号"})
		return
	}

	// 查询要删除的用户
	var user User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	// 开始事务
	tx := db.Begin()

	// 删除用户相关的消息
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

	currentUserID := getCurrentUserID(c)

	// 创建消息
	message := Message{
		SenderID:   currentUserID,
		ReceiverID: req.ReceiverID,
		Content:    req.Content,
		SentAt:     time.Now(),
	}

	if err := db.Create(&message).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送消息失败"})
		return
	}

	// 预加载关联数据
	db.Preload("Sender").Preload("Receiver").First(&message, message.ID)

	// 异步进行语法检查
	go checkGrammar(currentUserID, message)

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
		SentAt:     time.Now(),
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

// 获取语法错误记录
func getGrammarErrors(c *gin.Context) {
	currentUserID := getCurrentUserID(c)

	var errors []GrammarError
	if err := db.Where("user_id = ?", currentUserID).
		Order("created_at DESC").
		Preload("Message").
		Find(&errors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取记录失败"})
		return
	}

	c.JSON(http.StatusOK, errors)
}

// 删除语法错误记录
func deleteGrammarError(c *gin.Context) {
	errorID := c.Param("id")
	currentUserID := getCurrentUserID(c)

	id, err := strconv.Atoi(errorID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的ID"})
		return
	}

	// 确保只能删除自己的记录
	result := db.Where("id = ? AND user_id = ?", id, currentUserID).Delete(&GrammarError{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
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
