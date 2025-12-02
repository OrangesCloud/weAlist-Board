package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORS returns a middleware that handles CORS
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// 허용된 Origin 목록
		allowedOrigins := []string{
			"https://wealist.co.kr",
			"http://localhost:5173",
			"http://localhost:3000",
		}
		
		// Origin 검증 및 동적 설정
		allowed := false
		
		// CloudFront 도메인 패턴 매칭
		if strings.HasSuffix(origin, ".cloudfront.net") && strings.HasPrefix(origin, "https://") {
			allowed = true
		} else {
			// 명시적으로 허용된 Origin 확인
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					allowed = true
					break
				}
			}
		}
		
		// 허용된 Origin인 경우 CORS 헤더 설정
		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
			c.Writer.Header().Set("Access-Control-Max-Age", "43200")
		}
		
		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
