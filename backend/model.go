package main

import (
	"time"

	"gorm.io/gorm"
)

// 语法错误类型常量
const (
	ErrorTypePragmalinguistic = "语言语用失误" // 语言语用失误
	ErrorTypeSociopragmatic   = "社会语用失误" // 社会语用失误
)

// 用户角色常量
const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Username  string         `json:"username" gorm:"uniqueIndex;not null"`
	Password  string         `json:"-" gorm:"not null"` // 密码不返回给前端
	Country   string         `json:"country" gorm:"not null"`
	Role      string         `json:"role" gorm:"default:user"` // user 或 admin
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
	LLMSuggestion  string    `json:"llm_suggestion" gorm:"type:text"`                                    // 🔧 确保是 text 类型
	LLMExplanation string    `json:"llm_explanation" gorm:"type:text"`                                   // 🔧 确保是 text 类型
	ErrorType      string    `json:"error_type" gorm:"type:varchar(50);not null;default:'语言语用失误';index"` // 🔧 更新默认值
	CreatedAt      time.Time `json:"created_at"`

	User User `json:"user" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (GrammarError) TableName() string {
	return "grammar_errors"
}

// BeforeCreate 创建前的钩子
func (ge *GrammarError) BeforeCreate(tx *gorm.DB) error {
	// 如果没有设置错误类型,默认为语言语用失误
	if ge.ErrorType == "" {
		ge.ErrorType = ErrorTypePragmalinguistic
	}
	return nil
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

// 管理员相关请求结构
type UserManageRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Action string `json:"action" binding:"required"` // "delete", "promote", "demote"
}

// 用户列表响应
type UserListResponse struct {
	Users []User `json:"users"`
	Total int64  `json:"total"`
}

// 检查用户是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// 检查用户是否为普通用户
func (u *User) IsUser() bool {
	return u.Role == RoleUser
}

type WSMessage struct {
	Type      string      `json:"type"` // "message", "grammar_check", "online", "offline"
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}
