package admin

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware protects admin endpoints with a static token.
// Provide token via X-Admin-Token header (preferred) or Authorization: Bearer <token>.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := strings.TrimSpace(os.Getenv("ADMIN_TOKEN"))
		if expected == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "admin token is not configured"})
			c.Abort()
			return
		}

		token := strings.TrimSpace(c.GetHeader("X-Admin-Token"))
		if token == "" {
			authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
			if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
				token = strings.TrimSpace(authHeader[7:])
			}
		}
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing admin token"})
			c.Abort()
			return
		}

		if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid admin token"})
			c.Abort()
			return
		}

		sum := sha256.Sum256([]byte(token))
		c.Set("adminOperatorSource", "admin_console")
		c.Set("adminOperatorID", hex.EncodeToString(sum[:8]))
		c.Next()
	}
}
