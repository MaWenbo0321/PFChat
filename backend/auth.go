package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// 注册
func register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Country = strings.ToUpper(strings.TrimSpace(req.Country))
	if utf8.RuneCountInString(req.Username) < 2 || utf8.RuneCountInString(req.Username) > 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名长度必须为2到20个字符"})
		return
	}
	if !isValidUserCountry(req.Country) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的国家或地区"})
		return
	}

	// 检查用户名是否已存在
	var existUser User
	err := db.Where("username = ?", req.Username).First(&existUser).Error
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查用户名失败"})
		return
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// 创建用户
	user := User{
		Username: req.Username,
		Password: string(hashedPassword),
		Country:  req.Country,
		Role:     RoleUser, // 默认为普通用户
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建用户失败"})
		return
	}

	// 生成 token
	token, err := generateToken(user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User:  user,
	})
}

func isValidUserCountry(country string) bool {
	switch strings.ToUpper(strings.TrimSpace(country)) {
	case "CN", "TW", "HK", "SG", "MY", "JP", "KR", "MN", "FR", "DE",
		"US", "GB", "CA", "AU", "NZ", "NG", "BR", "ZA", "IN", "MX", "OTHER":
		return true
	default:
		return false
	}
}

// 登录
func login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Username = strings.TrimSpace(req.Username)

	// 查找用户
	var user User
	if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// Bot 用户不允许登录
	if user.Role == RoleBot {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 生成 token
	token, err := generateToken(user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User:  user,
	})
}

// 生成 JWT token
func generateToken(userID uint, username, role string) (string, error) {
	jwtSecret, err := getJWTSecret()
	if err != nil {
		return "", err
	}
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// 验证 token 中间件
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证token"})
			c.Abort()
			return
		}

		// 移除 "Bearer " 前缀
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token格式错误"})
			c.Abort()
			return
		}

		claims, err := parseTokenClaims(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "token无效或已过期"})
			c.Abort()
			return
		}

		// Token 中的角色可能在签发后已被管理员修改，用户也可能已被删除。
		// 每次请求以数据库中的当前状态为准，避免旧 Token 保留过期权限。
		var user User
		if err := db.First(&user, claims.UserID).Error; err != nil || user.Role == RoleBot {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在或不可登录"})
			c.Abort()
			return
		}

		// 将用户信息存入上下文
		c.Set("userID", user.ID)
		c.Set("username", user.Username)
		c.Set("role", user.Role)
		c.Next()
	}
}

func parseTokenClaims(tokenString string) (*Claims, error) {
	jwtSecret, err := getJWTSecret()
	if err != nil {
		return nil, err
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) { return jwtSecret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

func getJWTSecret() ([]byte, error) {
	secret, err := requireEnv("JWT_SECRET")
	if err != nil {
		return nil, err
	}
	if len([]byte(secret)) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	return []byte(secret), nil
}

// 管理员权限中间件
func adminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未获取到用户角色信息"})
			c.Abort()
			return
		}

		roleName, ok := role.(string)
		if !ok || roleName != RoleAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// 获取当前用户ID
func getCurrentUserID(c *gin.Context) uint {
	userID, ok := c.Get("userID")
	if !ok {
		return 0
	}
	id, _ := userID.(uint)
	return id
}
