package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/models"
	"github.com/tachigo/tachigo/internal/services"
)

const claimsKey = "claims"

// JWTAuth validates Bearer token and stores claims in context.
func JWTAuth(authSvc *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{"success": false, "error": "authorization header required"})
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := authSvc.ValidateAccessToken(token)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"success": false, "error": "invalid or expired token"})
			return
		}

		c.Set(claimsKey, claims)
		c.Next()
	}
}

// MustClaims returns the JWT claims stored by JWTAuth middleware.
// Panics if called outside an authenticated route.
func MustClaims(c *gin.Context) *services.Claims {
	v, _ := c.Get(claimsKey)
	return v.(*services.Claims)
}

// RequireRole ensures the authenticated user has one of the allowed roles.
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	allowed := make(map[models.UserRole]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		claims := MustClaims(c)
		if _, ok := allowed[claims.Role]; !ok {
			c.AbortWithStatusJSON(403, gin.H{"success": false, "error": "forbidden"})
			return
		}
		c.Next()
	}
}
