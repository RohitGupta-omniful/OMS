package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		expectedToken := "secret-token"

		if !strings.HasPrefix(authHeader, "Bearer ") || strings.TrimPrefix(authHeader, "Bearer ") != expectedToken {
			c.JSON(http.StatusUnauthorized, gin.H{
				"is_valid": false,
				"message":  "Unauthorized request",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
