package middleware

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	maxBodySize = 1024 * 10
)

type loggerResponseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r loggerResponseBodyWriter) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func sanitizeBody(body string) string {
	result := body
	sensitivePatterns := []struct {
		pattern     string
		replacement string
	}{
		{`"password"\s*:\s*"[^"]*"`, `"password":"***"`},
		{`"token"\s*:\s*"[^"]*"`, `"token":"***"`},
		{`"access_token"\s*:\s*"[^"]*"`, `"access_token":"***"`},
		{`"refresh_token"\s*:\s*"[^"]*"`, `"refresh_token":"***"`},
		{`"authorization"\s*:\s*"[^"]*"`, `"authorization":"***"`},
		{`"api_key"\s*:\s*"[^"]*"`, `"api_key":"***"`},
		{`"secret"\s*:\s*"[^"]*"`, `"secret":"***"`},
		{`"apikey"\s*:\s*"[^"]*"`, `"apikey":"***"`},
		{`"apisecret"\s*:\s*"[^"]*"`, `"apisecret":"***"`},
	}

	for _, p := range sensitivePatterns {
		re := regexp.MustCompile(p.pattern)
		result = re.ReplaceAllString(result, p.replacement)
	}

	return result
}

func readRequestBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "application/json") &&
		!strings.Contains(contentType, "application/x-www-form-urlencoded") &&
		!strings.Contains(contentType, "text/") {
		return "[비텍스트 유형, 생략됨]"
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "[요청 본문 읽기 실패]"
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var logBodyBytes []byte
	if len(bodyBytes) > maxBodySize {
		logBodyBytes = bodyBytes[:maxBodySize]
	} else {
		logBodyBytes = bodyBytes
	}

	bodyStr := string(logBodyBytes)
	if len(bodyBytes) > maxBodySize {
		bodyStr += "... [내용이 너무 길어 잘렸습니다]"
	}

	return sanitizeBody(bodyStr)
}

// RequestID middleware adds a unique request ID to the context
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get request ID from header or generate a new one
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		safeRequestID := secutils.SanitizeForLog(requestID)
		// Set request ID in header
		c.Header("X-Request-ID", requestID)

		// Set request ID in context
		c.Set(types.RequestIDContextKey.String(), requestID)

		// Set logger in context
		requestLogger := logger.GetLogger(c)
		requestLogger = requestLogger.WithField("request_id", safeRequestID)
		c.Set(types.LoggerContextKey.String(), requestLogger)

		// Set request ID in the global context for logging
		c.Request = c.Request.WithContext(
			context.WithValue(
				context.WithValue(c.Request.Context(), types.RequestIDContextKey, requestID),
				types.LoggerContextKey, requestLogger,
			),
		)

		c.Next()
	}
}

// Logger middleware logs request details with request ID, input and output
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		var requestBody string
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			requestBody = readRequestBody(c)
		}

		responseBody := &bytes.Buffer{}
		responseWriter := &loggerResponseBodyWriter{
			ResponseWriter: c.Writer,
			body:           responseBody,
		}
		c.Writer = responseWriter

		// Process request
		c.Next()

		// Get request ID from context
		requestID, exists := c.Get(types.RequestIDContextKey.String())
		requestIDStr := "unknown"
		if exists {
			if idStr, ok := requestID.(string); ok && idStr != "" {
				requestIDStr = idStr
			}
		}
		safeRequestID := secutils.SanitizeForLog(requestIDStr)

		// Calculate latency
		latency := time.Since(start)

		// Get client IP and status code
		clientIP := c.ClientIP()
		statusCode := c.Writer.Status()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + raw
		}

		responseBodyStr := ""
		if responseBody.Len() > 0 {
			contentType := c.Writer.Header().Get("Content-Type")
			if strings.Contains(contentType, "application/json") ||
				strings.Contains(contentType, "text/") {
				bodyBytes := responseBody.Bytes()
				if len(bodyBytes) > maxBodySize {
					responseBodyStr = string(bodyBytes[:maxBodySize]) + "... [내용이 너무 길어 잘렸습니다]"
				} else {
					responseBodyStr = string(bodyBytes)
				}
				responseBodyStr = sanitizeBody(responseBodyStr)
			} else {
				responseBodyStr = "[비텍스트 유형, 생략됨]"
			}
		}

		logMsg := logger.GetLogger(c)
		logMsg = logMsg.WithFields(map[string]interface{}{
			"request_id":  safeRequestID,
			"method":      method,
			"path":        secutils.SanitizeForLog(path),
			"status_code": statusCode,
			"size":        c.Writer.Size(),
			"latency":     latency.String(),
			"client_ip":   secutils.SanitizeForLog(clientIP),
		})

		if requestBody != "" {
			logMsg = logMsg.WithField("request_body", secutils.SanitizeForLog(requestBody))
		}

		if responseBodyStr != "" {
			logMsg = logMsg.WithField("response_body", secutils.SanitizeForLog(responseBodyStr))
		}
		logMsg.Info()
	}
}
