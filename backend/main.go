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

			// 语法错误记录（扩展功能）
			auth.GET("/grammar-errors", getGrammarErrors)                        // 获取错误记录(支持按类型筛选)
			auth.DELETE("/grammar-errors/:id", deleteGrammarError)               // 删除单条记录
			auth.POST("/grammar-errors/batch-delete", batchDeleteGrammarErrors)  // 批量删除
			auth.DELETE("/grammar-errors/clear/:type", clearGrammarErrorsByType) // 按类型清空
			auth.PUT("/grammar-errors/:id/type", updateGrammarErrorType)         // 更新错误类型

			// 管理员专用路由
			admin := auth.Group("/admin")
			admin.Use(adminMiddleware())
			{
				// 用户管理
				admin.GET("/users", getAllUsersForAdmin)
				admin.DELETE("/users/:id", deleteUser)
				admin.PUT("/users/:id/role", updateUserRole)

				// 系统统计
				admin.GET("/stats", getUserStats)
			}
		}
	}

	// WebSocket 路由
	r.GET("/ws", func(c *gin.Context) {
		serveWs(hub, c)
	})

	log.Println("Server starting on :8080")
	r.Run(":8080")
}

func initDB() {
	var err error
	dsn := "im_user:im_password@tcp(127.0.0.1:3306)/im_system?charset=utf8mb4&parseTime=True&loc=Local"
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 自动迁移
	db.AutoMigrate(&User{}, &Message{}, &GrammarError{})

	// 创建默认管理员账号（如果不存在）
	createDefaultAdmin()

	log.Println("Database connected and migrated")
}

// 创建默认管理员账号
func createDefaultAdmin() {
	var adminUser User

	// 检查是否已存在管理员账号
	if err := db.Where("role = ?", RoleAdmin).First(&adminUser).Error; err != nil {
		// 不存在管理员，创建默认管理员
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
			log.Println("⚠️  Please change the default admin password after first login!")
		}
	}
}

// CORS 中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 处理预检请求
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
