package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"google-ai-proxy/internal/auth"
	"google-ai-proxy/internal/db"
)

// UserAuthMiddleware 用户认证中间件
func UserAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "请先登录"})
			c.Abort()
			return
		}

		// 移除可能的 "Bearer " 前缀
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		claims, err := auth.ValidateUserToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "登录已过期，请重新登录"})
			c.Abort()
			return
		}

		// 验证用户是否存在
		var user db.User
		if err := db.DB.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
			c.Abort()
			return
		}

		if user.Status != "" && user.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "账号已被禁用"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Next()
	}
}
