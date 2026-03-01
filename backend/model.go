package main

import (
	"time"

	"gorm.io/gorm"
)

// 语用错误类型常量 - 5种分类
const (
	ErrorTypePragmalinguistic       = "语用语言失误"        // Pragmalinguistic failure (improvable)
	ErrorTypeSociopragmatic         = "社会语用失误"        // Sociopragmatic failure (improvable)
	ErrorTypeSeverePragmalinguistic = "严重语用语言失误"      // Severe pragmalinguistic failure (problematic)
	ErrorTypeSevereSociopragmatic   = "严重社会语用失误"      // Severe sociopragmatic failure (problematic)
	ErrorTypeBothFailure            = "语用语言失误和社会语用失误" // Both failures (problematic)
)

// AllValidErrorTypes 所有有效的错误类型
var AllValidErrorTypes = []string{
	ErrorTypePragmalinguistic,
	ErrorTypeSociopragmatic,
	ErrorTypeSeverePragmalinguistic,
	ErrorTypeSevereSociopragmatic,
	ErrorTypeBothFailure,
}

// IsValidErrorType 检查错误类型是否有效
func IsValidErrorType(errorType string) bool {
	for _, t := range AllValidErrorTypes {
		if t == errorType {
			return true
		}
	}
	return false
}

// IsProblematicErrorType 判断错误类型是否属于 "problematic" 级别
func IsProblematicErrorType(errorType string) bool {
	return errorType == ErrorTypeSeverePragmalinguistic ||
		errorType == ErrorTypeSevereSociopragmatic ||
		errorType == ErrorTypeBothFailure
}

// 用户角色常量
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Username  string         `json:"username" gorm:"type:varchar(191);uniqueIndex;not null"`
	Password  string         `json:"-" gorm:"type:varchar(255);not null"`
	Country   string         `json:"country" gorm:"type:varchar(10);not null"`
	Role      string         `json:"role" gorm:"type:varchar(20);default:user"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// Message 消息模型
type Message struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	SenderID   uint      `json:"sender_id" gorm:"not null;index"`
	ReceiverID uint      `json:"receiver_id" gorm:"not null;index"`
	Content    string    `json:"content" gorm:"type:text;not null"`
	IsRead     bool      `json:"is_read" gorm:"default:false"`
	CreatedAt  time.Time `json:"created_at"`

	// 关联
	Sender   User `json:"sender" gorm:"foreignKey:SenderID"`
	Receiver User `json:"receiver" gorm:"foreignKey:ReceiverID"`
}

// GrammarError 语法错误记录模型
type GrammarError struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	UserID         uint      `json:"user_id" gorm:"not null;index"`
	MessageID      uint      `json:"message_id" gorm:"index"`
	OriginalText   string    `json:"original_text" gorm:"type:text;not null"`
	LLMSuggestion  string    `json:"llm_suggestion" gorm:"type:text"`
	LLMExplanation string    `json:"llm_explanation" gorm:"type:text"`
	ErrorType      string    `json:"error_type" gorm:"type:varchar(50);not null;default:'语用语言失误';index"`
	CreatedAt      time.Time `json:"created_at"`

	User User `json:"user" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (GrammarError) TableName() string {
	return "grammar_errors"
}

// BeforeCreate 创建前的钩子
func (ge *GrammarError) BeforeCreate(tx *gorm.DB) error {
	if ge.ErrorType == "" {
		ge.ErrorType = ErrorTypePragmalinguistic
	}
	return nil
}

// AIChatHistory AI对话历史
type AIChatHistory struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"user_id" gorm:"not null;index"`
	ChatUserID uint      `json:"chat_user_id" gorm:"index"` // 当前聊天对象ID (上下文)
	Role       string    `json:"role" gorm:"type:varchar(20);not null"`
	Content    string    `json:"content" gorm:"type:text;not null"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AIChatHistory) TableName() string {
	return "ai_chat_history"
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
	Username string `json:"username" binding:"required,min=2,max=20"`
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

type UserManageRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Action string `json:"action" binding:"required"`
}

type UserListResponse struct {
	Users []User `json:"users"`
	Total int64  `json:"total"`
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) IsUser() bool {
	return u.Role == RoleUser
}

type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}
