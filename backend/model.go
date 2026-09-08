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
	ErrorTypeNoIssue                = "无明显语用失误"       // Session feedback with no clear pragmatic issue
)

// AllValidErrorTypes 所有有效的错误类型
var AllValidErrorTypes = []string{
	ErrorTypePragmalinguistic,
	ErrorTypeSociopragmatic,
	ErrorTypeSeverePragmalinguistic,
	ErrorTypeSevereSociopragmatic,
	ErrorTypeBothFailure,
	ErrorTypeNoIssue,
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
	RoleBot   = "bot"
)

// 语用错误来源
const (
	ErrorSourceUser = "user"
	ErrorSourceLLM  = "llm"
)

// 对话模式常量
const (
	ModeUserL2 = "user_l2" // 用户使用第二语言，LLM作为母语者
	ModeLLML2  = "llm_l2"  // LLM使用第二语言，模拟非流利说话者
)

// 会话反馈模式常量
const (
	FeedbackComplete = "complete" // 完整对话：用户手动结束或10分钟超时
	FeedbackRounds5  = "rounds_5" // 逐轮反馈，用户手动结束；保留旧值以兼容已有会话
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
	SessionID  uint      `json:"session_id" gorm:"index;default:0"`
	Role       string    `json:"role" gorm:"type:varchar(20);default:'user'"` // "user" or "llm"
	CreatedAt  time.Time `json:"created_at"`

	// 关联
	Sender   User `json:"sender" gorm:"foreignKey:SenderID"`
	Receiver User `json:"receiver" gorm:"foreignKey:ReceiverID"`
}

// ConversationSession 对话会话
type ConversationSession struct {
	ID                   uint      `json:"id" gorm:"primaryKey"`
	UserID               uint      `json:"user_id" gorm:"not null;index"`
	BotUserID            uint      `json:"bot_user_id" gorm:"not null"`
	RelationshipType     string    `json:"relationship_type" gorm:"type:varchar(50)"`
	Topic                string    `json:"topic" gorm:"type:varchar(100)"`
	Mode                 string    `json:"mode" gorm:"type:varchar(20)"` // ModeUserL2 or ModeLLML2
	TargetLanguage       string    `json:"target_language" gorm:"type:varchar(10)"`
	LLMRoleID            string    `json:"llm_role_id" gorm:"type:varchar(50);default:'aiko'"`
	LLMCountry           string    `json:"llm_country" gorm:"type:varchar(10);index"`
	LLMNativeLanguage    string    `json:"llm_native_language" gorm:"type:varchar(120)"`
	LLMNameZH            string    `json:"llm_name_zh" gorm:"type:varchar(100)"`
	LLMNameEN            string    `json:"llm_name_en" gorm:"type:varchar(120)"`
	LLMAge               int       `json:"llm_age"`
	LLMGenderZH          string    `json:"llm_gender_zh" gorm:"type:varchar(30)"`
	LLMGenderEN          string    `json:"llm_gender_en" gorm:"type:varchar(30)"`
	LLMPersonalityZH     string    `json:"llm_personality_zh" gorm:"type:text"`
	LLMPersonalityEN     string    `json:"llm_personality_en" gorm:"type:text"`
	LLMBackgroundZH      string    `json:"llm_background_zh" gorm:"type:text"`
	LLMBackgroundEN      string    `json:"llm_background_en" gorm:"type:text"`
	FeedbackMode         string    `json:"feedback_mode" gorm:"type:varchar(20)"` // FeedbackComplete or FeedbackRounds5
	AISuggestionsEnabled bool      `json:"ai_suggestions_enabled" gorm:"default:true"`
	RoundCount           int       `json:"round_count" gorm:"default:0"`
	IsActive             bool      `json:"is_active" gorm:"default:true"`
	SummaryFeedback      string    `json:"summary_feedback" gorm:"type:text"` // 会话结束汇总
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (ConversationSession) TableName() string {
	return "conversation_sessions"
}

// GrammarError 语法错误记录模型
type GrammarError struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	UserID              uint      `json:"user_id" gorm:"not null;index"`
	SessionID           uint      `json:"session_id" gorm:"index;default:0"`
	SessionMode         string    `json:"session_mode" gorm:"type:varchar(20);index"`
	MessageID           uint      `json:"message_id" gorm:"index"`
	SourceRole          string    `json:"source_role" gorm:"type:varchar(20);not null;default:'user';index"`
	OriginalText        string    `json:"original_text" gorm:"type:text;not null"`
	ConversationSummary string    `json:"conversation_summary" gorm:"type:text"`
	LLMIntendedMeaning  string    `json:"llm_intended_meaning" gorm:"type:text"`
	LLMSuggestion       string    `json:"llm_suggestion" gorm:"type:text"`
	LLMExplanation      string    `json:"llm_explanation" gorm:"type:text"`
	ErrorType           string    `json:"error_type" gorm:"type:varchar(50);not null;default:'语用语言失误';index"`
	OverallEvaluation   string    `json:"overall_evaluation" gorm:"type:varchar(20);index"`
	IssueCount          int       `json:"issue_count" gorm:"not null;default:1"`
	CreatedAt           time.Time `json:"created_at"`

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
	if ge.SourceRole == "" {
		ge.SourceRole = ErrorSourceUser
	}
	if ge.IssueCount <= 0 && ge.ErrorType != ErrorTypeNoIssue {
		ge.IssueCount = 1
	}
	if ge.SessionMode != ModeUserL2 && ge.SessionMode != ModeLLML2 {
		switch ge.SourceRole {
		case ErrorSourceLLM:
			ge.SessionMode = ModeLLML2
		case ErrorSourceUser:
			ge.SessionMode = ModeUserL2
		}
	}
	return nil
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

type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}
