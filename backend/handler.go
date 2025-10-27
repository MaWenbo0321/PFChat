package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// 获取用户列表
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

	// 只能删除自己发送的消息
	if message.SenderID != currentUserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权删除此消息"})
		return
	}

	// 软删除消息（更新为已删除标记）
	// 这里使用硬删除，如果需要软删除可以添加 deleted_at 字段
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
