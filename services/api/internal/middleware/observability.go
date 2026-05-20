package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	RequestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
)

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := ""
		if c.Request != nil {
			requestID = c.Request.Header.Get(RequestIDHeader)
		}
		if !isSafeRequestID(requestID) {
			requestID = newRequestID()
		}

		c.Set(requestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)
		c.Next()
	}
}

func RequestIDFromGin(c *gin.Context) string {
	if c == nil {
		return ""
	}
	value, ok := c.Get(requestIDKey)
	if !ok {
		return ""
	}
	requestID, ok := value.(string)
	if !ok {
		return ""
	}
	return requestID
}

func StructuredRequestLogger(logger *log.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = log.Default()
	}

	return func(c *gin.Context) {
		start := time.Now()
		method := ""
		path := ""
		if c.Request != nil {
			method = c.Request.Method
			if c.Request.URL != nil {
				path = c.Request.URL.Path
			}
		}

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = path
		}
		logger.Printf(
			"event=http_request request_id=%s method=%s route=%s path=%s status=%d duration_ms=%d client_ip=%s errors=%d",
			RequestIDFromGin(c),
			safeLogToken(method),
			safeLogToken(route),
			safeLogToken(path),
			c.Writer.Status(),
			time.Since(start).Milliseconds(),
			safeLogToken(c.ClientIP()),
			len(c.Errors),
		)
	}
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format("20060102150405.000000000")))
	}
	return hex.EncodeToString(b[:])
}

func isSafeRequestID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if r >= 'a' && r <= 'z' ||
			r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' ||
			r == '-' || r == '_' || r == '.' || r == ':' {
			continue
		}
		return false
	}
	return true
}

func safeLogToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	value = strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r', '"', '\'':
			return '_'
		default:
			return r
		}
	}, value)
	if len(value) > 256 {
		return value[:256]
	}
	return value
}
