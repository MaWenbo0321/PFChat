package main

import (
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

	// 设置全局 hub
	globalHub = hub

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

			// 消息相关
			auth.GET("/messages/:userId", getMessages)
			auth.POST("/messages", sendMessage)
			auth.POST("/messages/check", checkMessageBeforeSend)
			auth.DELETE("/messages/:id", deleteMessage)
			auth.DELETE("/messages/clear/:userId", clearChatHistory)
			auth.DELETE("/messages/mine/:userId", deleteMyMessages)

			// 🆕 实时检测接口 (Grammarly 风格)
			auth.POST("/messages/realtime-check", realtimeCheck)

			// 🆕 获取消息关联的语用错误 (接收方使用)
			auth.POST("/messages/errors", getMessageErrorsByIds)

			// 🆕 AI Chat 接口
			auth.POST("/ai/chat", aiChat)

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
	r.Run("0.0.0.0:8080")
}

func initDB() {
	var err error
	dsn := "im_user:im_password@tcp(:3306)/im_system?charset=utf8mb4&parseTime=True&loc=Local"
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 自动迁移 - 🆕 新增 AIChatHistory
	db.AutoMigrate(&User{}, &Message{}, &GrammarError{}, &AIChatHistory{})

	// 创建默认管理员账号
	createDefaultAdmin()

	log.Println("Database connected and migrated")
}

func createDefaultAdmin() {
	var adminUser User
	if err := db.Where("role = ?", RoleAdmin).First(&adminUser).Error; err != nil {
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
