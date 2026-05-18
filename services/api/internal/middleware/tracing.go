package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func Tracing(tracer trace.Tracer) gin.HandlerFunc {
	return func(c *gin.Context) {
		if tracer == nil || c.Request == nil {
			c.Next()
			return
		}

		method := c.Request.Method
		ctx, span := tracer.Start(c.Request.Context(), "HTTP "+method)
		defer span.End()
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := c.Writer.Status()
		span.SetName(method + " " + route)
		span.SetAttributes(
			attribute.String("http.request.method", method),
			attribute.String("http.route", route),
			attribute.Int("http.response.status_code", status),
			attribute.String("request_id", RequestIDFromGin(c)),
			attribute.Int("gin.error_count", len(c.Errors)),
		)
		if status >= http.StatusInternalServerError || len(c.Errors) > 0 {
			description := http.StatusText(status)
			if len(c.Errors) > 0 && status < http.StatusInternalServerError {
				description = "gin errors present"
			}
			span.SetStatus(codes.Error, description)
		}
	}
}
