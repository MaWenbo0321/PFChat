package main

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func main() {
	// 初始化数据库
	initDB()

	// 初始化 WebSocket Hub
	hub := newHub()
	go hub.run()

	// 初始化路由
	r := gin.Default()

	// CORS 中间件
	r.Use(corsMiddleware())

	// API 路由
	api := r.Group("/api")
	{
		// 用户认证
		api.POST("/register", register)
		api.POST("/login", login)

		// 需要认证的路由
		auth := api.Group("/")
		auth.Use(authMiddleware())
		{
			// 用户信息
			auth.GET("/users", getUsers)
			auth.GET("/users/:id", getUserInfo)

			// 获取消息关联的语用错误
			auth.POST("/messages/errors", getMessageErrorsByIds)

			// 会话管理接口
			auth.POST("/sessions", createSession)
			auth.GET("/sessions/active", getActiveSession)
			auth.POST("/sessions/:id/end", endSession)
			auth.GET("/sessions/:id/messages", getSessionMessages)
			auth.GET("/sessions/:id/feedback", getSessionFeedback)

			// LLM 对话接口
			auth.POST("/llm/message", sendLLMMessage)

			// LLM Bot 信息接口
			auth.GET("/bot/info", getLLMBotInfo)

			// 语法错误记录
			auth.GET("/grammar-errors", getGrammarErrors)
			auth.DELETE("/grammar-errors/:id", deleteGrammarError)
			auth.POST("/grammar-errors/batch-delete", batchDeleteGrammarErrors)
			auth.DELETE("/grammar-errors/clear/:type", clearGrammarErrorsByType)
			auth.PUT("/grammar-errors/:id/type", updateGrammarErrorType)

			// 管理员专用路由
			admin := auth.Group("/admin")
			admin.Use(adminMiddleware())
			{
				admin.GET("/users", getAllUsersForAdmin)
				admin.DELETE("/users/:id", deleteUser)
				admin.PUT("/users/:id/role", updateUserRole)
				admin.GET("/stats", getUserStats)
			}
		}
	}

	// WebSocket 路由
	r.GET("/ws", func(c *gin.Context) {
		serveWs(hub, c)
	})

	log.Println("Server starting on :8080")
	if err := r.Run("0.0.0.0:8080"); err != nil {
		log.Fatal("Server failed:", err)
	}
}

func initDB() {
	var err error
	dsn := "im_user:im_password@tcp(:3306)/im_system?charset=utf8mb4&parseTime=True&loc=Local"
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 自动迁移
	if err := db.AutoMigrate(&User{}, &Message{}, &GrammarError{}, &ConversationSession{}); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// 创建默认管理员账号
	createDefaultAdmin()

	// 创建 LLM Bot 用户
	createLLMBotUser()

	log.Println("Database connected and migrated")
}

func createLLMBotUser() {
	var botUser User
	err := db.Where("role = ?", RoleBot).First(&botUser).Error
	if err == nil {
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Failed to query LLM bot user: %v", err)
		return
	}
	{
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("llmbot_not_login"), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash bot password: %v", err)
			return
		}
		bot := User{
			Username: "LLM助手",
			Password: string(hashedPassword),
			Country:  "OTHER",
			Role:     RoleBot,
		}
		if err := db.Create(&bot).Error; err != nil {
			log.Printf("Failed to create LLM bot user: %v", err)
		} else {
			log.Printf("LLM bot user created with ID %d", bot.ID)
		}
	}
}

func createDefaultAdmin() {
	var adminUser User
	err := db.Where("role = ?", RoleAdmin).First(&adminUser).Error
	if err == nil {
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("Failed to query default admin: %v", err)
		return
	}
	{
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123456"), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash admin password: %v", err)
			return
		}

		defaultAdmin := User{
			Username: "admin",
			Password: string(hashedPassword),
			Country:  "CN",
			Role:     RoleAdmin,
		}

		if err := db.Create(&defaultAdmin).Error; err != nil {
			log.Printf("Failed to create default admin: %v", err)
		} else {
			log.Println("Default admin account created - Username: admin, Password: admin123456")
		}
	}
}

// CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 🔧 在生产环境中，应该指定具体的源
		origin := c.GetHeader("Origin")
		if origin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Upgrade, Connection")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
