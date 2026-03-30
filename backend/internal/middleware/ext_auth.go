package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/services"
)

const extClaimsKey = "ext_claims"

// ExtJWTAuth validates a Twitch Extension Bearer token and stores claims in context.
func ExtJWTAuth(extSvc *services.ExtensionService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(401, gin.H{"success": false, "error": "authorization header required"})
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := extSvc.VerifyExtJWT(token)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"success": false, "error": "invalid extension token"})
			return
		}

		c.Set(extClaimsKey, claims)
		c.Next()
	}
}

// MustExtClaims returns the Extension JWT claims stored by ExtJWTAuth middleware.
// Panics with a clear message if called outside an ext-authenticated route or
// if the stored value is not *services.ExtensionClaims.
func MustExtClaims(c *gin.Context) *services.ExtensionClaims {
	v, ok := c.Get(extClaimsKey)
	if !ok {
		panic("MustExtClaims: ext_claims not found in context — is ExtJWTAuth middleware applied?")
	}
	claims, ok := v.(*services.ExtensionClaims)
	if !ok {
		panic("MustExtClaims: ext_claims value is not *services.ExtensionClaims")
	}
	return claims
}
