package middleware

import (
	"crypto/subtle"

	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/config"
)

const tachiyaInternalSecretHeader = "X-Tachiya-Internal-Secret"

func TachiyaInternalAuth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := cfg.Internal.TachiyaSharedSecret
		actual := c.GetHeader(tachiyaInternalSecretHeader)

		if expected == "" || actual == "" || subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
			c.AbortWithStatusJSON(401, gin.H{"success": false, "error": "invalid internal secret"})
			return
		}

		c.Next()
	}
}
