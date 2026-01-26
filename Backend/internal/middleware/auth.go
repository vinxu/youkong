package middleware

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"youkong/internal/pkg/jwt"
	"youkong/internal/pkg/response"
)

const (
	AuthorizationHeader = "Authorization"
	BearerPrefix        = "Bearer "
	ContextUserIDKey    = "userID"
	ContextPhoneKey     = "phone"
)

func Auth(jwtManager *jwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		claims, err := jwtManager.ParseToken(tokenString)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				response.TokenExpired(c)
			} else {
				response.Unauthorized(c)
			}
			c.Abort()
			return
		}

		c.Set(ContextUserIDKey, claims.UserID)
		c.Set(ContextPhoneKey, claims.Phone)
		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	userID, _ := c.Get(ContextUserIDKey)
	if id, ok := userID.(string); ok {
		return id
	}
	return ""
}

func GetPhone(c *gin.Context) string {
	phone, _ := c.Get(ContextPhoneKey)
	if p, ok := phone.(string); ok {
		return p
	}
	return ""
}
