package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ============================================================================
// 用户相关处理函数
// ============================================================================

// 获取用户列表（排除当前用户和bot用户）
func getUsers(c *gin.Context) {
	currentUserID := getCurrentUserID(c)

	var users []User
	if err := db.Where("id != ? AND role != ?", currentUserID, RoleBot).Find(&users).Error; err != nil {
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

// ============================================================================
// 语法错误相关处理函数（已扩展支持错误类型）
// ============================================================================

// GetGrammarErrorsRequest 获取语法错误请求
type GetGrammarErrorsRequest struct {
	ErrorType   string `form:"error_type"`   // 错误类型筛选
	SessionMode string `form:"session_mode"` // user_l2 / llm_l2 / all
}

// GetGrammarErrorsResponse 获取语法错误响应
type GetGrammarErrorsResponse struct {
	Errors     []GrammarError         `json:"errors"`
	Statistics GrammarErrorStatistics `json:"statistics"`
}

// GrammarErrorStatistics 语法错误统计
type GrammarErrorStatistics struct {
	Total      int64          `json:"total"`
	ByType     map[string]int `json:"by_type"`
	TodayCount int64          `json:"today_count"`
	WeekCount  int64          `json:"week_count"`
}

// 获取当前用户的语法错误记录（支持按类型筛选）
func getGrammarErrors(c *gin.Context) {
	userID := getCurrentUserID(c)

	var req GetGrammarErrorsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 构建查询
	query := db.Where("user_id = ?", userID)

	// 按错误类型筛选
	if req.ErrorType != "" && req.ErrorType != "all" {
		if !IsValidErrorType(req.ErrorType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的错误类型"})
			return
		}
		query = query.Where("error_type = ?", req.ErrorType)
	}
	if req.SessionMode != "" && req.SessionMode != "all" {
		if !isValidSessionModeFilter(req.SessionMode) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的练习者类型"})
			return
		}
		query = query.Where("session_mode = ?", req.SessionMode)
	}

	// 获取错误记录
	var errors []GrammarError
	if err := query.Order("created_at DESC").Find(&errors).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取错误记录失败"})
		return
	}

	// 获取统计信息
	statistics, err := getGrammarErrorStatistics(userID, req.SessionMode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取错误统计失败"})
		return
	}

	c.JSON(http.StatusOK, GetGrammarErrorsResponse{
		Errors:     errors,
		Statistics: statistics,
	})
}

func isValidSessionModeFilter(mode string) bool {
	return mode == ModeUserL2 || mode == ModeLLML2
}

// 获取语法错误统计信息
func grammarErrorStatisticsQuery(userID uint, sessionMode string) *gorm.DB {
	query := db.Model(&GrammarError{}).Where("user_id = ?", userID)
	if isValidSessionModeFilter(sessionMode) {
		query = query.Where("session_mode = ?", sessionMode)
	}
	return query
}

func getGrammarErrorStatistics(userID uint, sessionMode string) (GrammarErrorStatistics, error) {
	var statistics GrammarErrorStatistics

	// 总数
	if err := grammarErrorStatisticsQuery(userID, sessionMode).Count(&statistics.Total).Error; err != nil {
		return statistics, err
	}

	// 按类型统计
	statistics.ByType = make(map[string]int)
	var typeStats []struct {
		ErrorType string
		Count     int
	}
	if err := grammarErrorStatisticsQuery(userID, sessionMode).
		Select("error_type, COUNT(*) as count").
		Group("error_type").
		Scan(&typeStats).Error; err != nil {
		return statistics, err
	}

	for _, stat := range typeStats {
		statistics.ByType[stat.ErrorType] = stat.Count
	}

	// 今日错误数
	if err := grammarErrorStatisticsQuery(userID, sessionMode).
		Where("DATE(created_at) = CURDATE()").
		Count(&statistics.TodayCount).Error; err != nil {
		return statistics, err
	}

	// 本周错误数
	if err := grammarErrorStatisticsQuery(userID, sessionMode).
		Where("YEARWEEK(created_at, 1) = YEARWEEK(CURDATE(), 1)").
		Count(&statistics.WeekCount).Error; err != nil {
		return statistics, err
	}

	return statistics, nil
}

// 删除语法错误记录
func deleteGrammarError(c *gin.Context) {
	userID := getCurrentUserID(c)
	errorID := c.Param("id")

	// 检查记录是否属于当前用户
	var grammarError GrammarError
	if err := db.Where("id = ? AND user_id = ?", errorID, userID).First(&grammarError).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}

	// 删除记录
	if err := db.Delete(&grammarError).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// 批量删除语法错误记录
func batchDeleteGrammarErrors(c *gin.Context) {
	userID := getCurrentUserID(c)

	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 只删除属于当前用户的记录
	result := db.Where("user_id = ? AND id IN ?", userID, req.IDs).Delete(&GrammarError{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "批量删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "批量删除成功",
		"deleted_count": result.RowsAffected,
	})
}

// 按类型清空语法错误记录
func clearGrammarErrorsByType(c *gin.Context) {
	userID := getCurrentUserID(c)
	errorType := c.Param("type")
	sessionMode := c.Query("session_mode")
	if sessionMode != "" && sessionMode != "all" && !isValidSessionModeFilter(sessionMode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的练习者类型"})
		return
	}

	query := db.Where("user_id = ?", userID)
	if sessionMode != "" && sessionMode != "all" {
		query = query.Where("session_mode = ?", sessionMode)
	}

	if errorType == "" || errorType == "all" {
		// 清空筛选范围内的全部记录
		result := query.Delete(&GrammarError{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "清空失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":       "清空成功",
			"deleted_count": result.RowsAffected,
		})
	} else {
		if !IsValidErrorType(errorType) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的错误类型"})
			return
		}
		// 清空指定类型的记录
		result := query.Where("error_type = ?", errorType).Delete(&GrammarError{})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "清空失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message":       "清空成功",
			"deleted_count": result.RowsAffected,
		})
	}
}

// 更新语法错误的类型
func updateGrammarErrorType(c *gin.Context) {
	userID := getCurrentUserID(c)
	errorID := c.Param("id")

	var req struct {
		ErrorType string `json:"error_type" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	if !IsValidErrorType(req.ErrorType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的错误类型"})
		return
	}

	// 检查记录是否属于当前用户
	var grammarError GrammarError
	if err := db.Where("id = ? AND user_id = ?", errorID, userID).First(&grammarError).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
		return
	}

	// 更新错误类型
	if err := db.Model(&grammarError).Update("error_type", req.ErrorType).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// ============================================================================
// 管理员专用处理函数
// ============================================================================

// 获取所有用户（管理员专用）
func getAllUsersForAdmin(c *gin.Context) {
	var users []User
	if err := db.Where("role <> ?", RoleBot).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users": users,
		"total": len(users),
	})
}

// 删除用户（管理员专用）
func deleteUser(c *gin.Context) {
	userID := c.Param("id")
	currentUserID := getCurrentUserID(c)

	id64, err := strconv.ParseUint(userID, 10, 64)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}
	id := uint(id64)

	// 不能删除自己
	if id == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除自己"})
		return
	}

	// 查询用户
	var user User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if user.Role == RoleBot {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除系统 Bot 用户"})
		return
	}

	// 使用事务删除用户及相关数据
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&GrammarError{}).Error; err != nil {
			return fmt.Errorf("删除用户语法记录: %w", err)
		}
		if err := tx.Where("sender_id = ? OR receiver_id = ?", id, id).Delete(&Message{}).Error; err != nil {
			return fmt.Errorf("删除用户消息: %w", err)
		}
		if err := tx.Where("user_id = ?", id).Delete(&ConversationSession{}).Error; err != nil {
			return fmt.Errorf("删除用户会话: %w", err)
		}
		// 用户关联数据已硬删除；用户本身也应硬删除，避免保留密码哈希并永久占用唯一用户名。
		if err := tx.Unscoped().Delete(&user).Error; err != nil {
			return fmt.Errorf("删除用户: %w", err)
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户及关联数据失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "用户删除成功",
		"deleted_user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}

// 修改用户角色（管理员专用）
func updateUserRole(c *gin.Context) {
	userID := c.Param("id")
	currentUserID := getCurrentUserID(c)

	id64, err := strconv.ParseUint(userID, 10, 64)
	if err != nil || id64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}
	id := uint(id64)

	// 不能修改自己的角色
	if id == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能修改自己的角色"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证角色值
	if req.Role != RoleUser && req.Role != RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色"})
		return
	}

	// 查询用户
	var user User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if user.Role == RoleBot {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能修改系统 Bot 用户角色"})
		return
	}

	// 更新角色
	if err := db.Model(&user).Update("role", req.Role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户角色失败"})
		return
	}

	user.Role = req.Role

	c.JSON(http.StatusOK, gin.H{
		"message": "用户角色更新成功",
		"user":    user,
	})
}

// 获取用户统计信息（管理员专用）
func getUserStats(c *gin.Context) {
	var stats struct {
		TotalUsers         int64 `json:"total_users"`
		AdminUsers         int64 `json:"admin_users"`
		RegularUsers       int64 `json:"regular_users"`
		TotalMessages      int64 `json:"total_messages"`
		TotalGrammarErrors int64 `json:"total_grammar_errors"`
	}

	queries := []*gorm.DB{
		db.Model(&User{}).Where("role <> ?", RoleBot).Count(&stats.TotalUsers),
		db.Model(&User{}).Where("role = ?", RoleAdmin).Count(&stats.AdminUsers),
		db.Model(&User{}).Where("role = ?", RoleUser).Count(&stats.RegularUsers),
		db.Model(&Message{}).Count(&stats.TotalMessages),
		db.Model(&GrammarError{}).Count(&stats.TotalGrammarErrors),
	}
	for _, query := range queries {
		if query.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取统计信息失败"})
			return
		}
	}

	c.JSON(http.StatusOK, stats)
}
