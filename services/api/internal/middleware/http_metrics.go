package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tachigo/tachigo/internal/metrics"
)

func HTTPMetrics(collector *metrics.Collector) gin.HandlerFunc {
	return func(c *gin.Context) {
		if collector == nil {
			c.Next()
			return
		}

		start := time.Now()
		c.Next()

		route := c.FullPath()
		collector.ObserveHTTPRequest(route, c.Writer.Status(), time.Since(start))
	}
}

func MetricsBearerGuard(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			c.Next()
			return
		}

		header := c.GetHeader("Authorization")
		got, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}

		c.Next()
	}
}
