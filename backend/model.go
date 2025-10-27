package main

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Username  string         `gorm:"unique;not null" json:"username"`
	Password  string         `gorm:"not null" json:"-"`
	Country   string         `gorm:"not null" json:"country"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Message 消息模型
type Message struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	SenderID   uint      `gorm:"not null;index" json:"sender_id"`
	ReceiverID uint      `gorm:"not null;index" json:"receiver_id"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	IsRead     bool      `gorm:"default:false" json:"is_read"`
	SentAt     time.Time `json:"sent_at"`
	CreatedAt  time.Time `json:"created_at"`

	// 关联
	Sender   User `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	Receiver User `gorm:"foreignKey:ReceiverID" json:"receiver,omitempty"`
}

// GrammarError 语法错误记录模型
type GrammarError struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index" json:"user_id"`
	MessageID      uint      `gorm:"not null" json:"message_id"`
	OriginalText   string    `gorm:"type:text;not null" json:"original_text"`
	LLMSuggestion  string    `gorm:"type:text;not null" json:"llm_suggestion"`
	LLMExplanation string    `gorm:"type:text" json:"llm_explanation"`
	CreatedAt      time.Time `json:"created_at"`

	// 关联
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Message Message `gorm:"foreignKey:MessageID" json:"message,omitempty"`
}

// WebSocket 消息结构
type WSMessage struct {
	Type      string      `json:"type"` // "message", "grammar_check", "online", "offline"
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// 语法检查结果
type GrammarCheckResult struct {
	HasError    bool   `json:"has_error"`
	Suggestion  string `json:"suggestion"`
	Explanation string `json:"explanation"`
	MessageID   uint   `json:"message_id"`
}

// 请求/响应结构
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,min=6"`
	Country  string `json:"country" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type SendMessageRequest struct {
	ReceiverID uint   `json:"receiver_id" binding:"required"`
	Content    string `json:"content" binding:"required"`
}
