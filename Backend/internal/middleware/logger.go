package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 读取请求体（仅对 POST/PUT 请求）
		var bodyBytes []byte
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 记录请求开始
		logger.Debug("request started",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("ip", c.ClientIP()),
			zap.String("content-type", c.ContentType()),
			zap.Int64("content-length", c.Request.ContentLength),
			zap.String("body", string(bodyBytes)),
		)

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.Int("status", status),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("latency", latency),
		}

		if userID := GetUserID(c); userID != "" {
			fields = append(fields, zap.String("user_id", userID))
		}

		// 如果是 500 错误，记录更多信息
		if status >= 500 {
			fields = append(fields,
				zap.String("content-type", c.ContentType()),
				zap.Int64("content-length", c.Request.ContentLength),
				zap.String("body", string(bodyBytes)),
			)
		}

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				logger.Error(e, fields...)
			}
		} else {
			if status >= 500 {
				logger.Error(path, fields...)
			} else if status >= 400 {
				logger.Warn(path, fields...)
			} else {
				logger.Info(path, fields...)
			}
		}
	}
}
