package middleware

import (
	"net/http"
	"strings"

	"pollster-backend/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const userIDKey = "userID"

// RequireAuth extracts and validates the Bearer JWT from the Authorization header.
// On success it stores the user's ObjectID in the Gin context under key "userID".
func RequireAuth(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		userID, err := authSvc.ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(userIDKey, userID)
		c.Next()
	}
}

// GetUserID retrieves the authenticated user's ObjectID from the Gin context.
// Panics if called outside an authenticated route (programming error).
func GetUserID(c *gin.Context) primitive.ObjectID {
	val, exists := c.Get(userIDKey)
	if !exists {
		panic("GetUserID called outside auth middleware")
	}
	return val.(primitive.ObjectID)
}
