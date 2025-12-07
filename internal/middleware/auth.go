// Package middleware provides HTTP middleware for the application.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Auth returns a middleware that validates JWT tokens
func Auth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Authorization header is required",
				},
				"message": "인증이 필요합니다",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid authorization header format",
				},
				"message": "잘못된 인증 헤더 형식입니다",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid or expired token",
				},
				"message": "유효하지 않거나 만료된 토큰입니다",
			})
			c.Abort()
			return
		}

		// Extract user ID from claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid token claims",
				},
				"message": "유효하지 않은 토큰 정보입니다",
			})
			c.Abort()
			return
		}

		// Extract user ID from claims (support multiple claim formats)
		var userIDStr string
		
		// Try "user_id" first (our format)
		if uid, ok := claims["user_id"].(string); ok {
			userIDStr = uid
		} else if sub, ok := claims["sub"].(string); ok {
			// Try "sub" (Google OAuth format)
			userIDStr = sub
		} else if uid, ok := claims["uid"].(string); ok {
			// Try "uid" (alternative format)
			userIDStr = uid
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "User ID not found in token",
				},
				"message": "토큰에서 사용자 ID를 찾을 수 없습니다",
			})
			c.Abort()
			return
		}

		// Parse user ID as UUID
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"code":    "UNAUTHORIZED",
					"message": "Invalid user ID format",
				},
				"message": "유효하지 않은 사용자 ID 형식입니다",
			})
			c.Abort()
			return
		}

		// Store user ID and JWT token in context for downstream use
		c.Set("user_id", userID)
		c.Set("jwtToken", tokenString)

		c.Next()
	}
}
