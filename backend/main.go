package main

import (
	"log"

	"github.com/gin-gonic/gin"
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

			// 消息相关
			auth.GET("/messages/:userId", getMessages)
			auth.POST("/messages", sendMessage)
			auth.DELETE("/messages/:id", deleteMessage)
			auth.DELETE("/messages/clear/:userId", clearChatHistory)
			auth.DELETE("/messages/mine/:userId", deleteMyMessages)

			// 语法错误记录
			auth.GET("/grammar-errors", getGrammarErrors)
			auth.DELETE("/grammar-errors/:id", deleteGrammarError)
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
	log.Println("Database connected and migrated")
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
