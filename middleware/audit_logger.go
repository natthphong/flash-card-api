package middleware

import (
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

var rePassword = regexp.MustCompile(`(?i)"password"\s*:\s*"(.*?)"`)

func AuditLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		logger := logz.NewLogger()
		reqID := c.Get("requestId")

		ct := strings.ToLower(c.Get("Content-Type"))

		if strings.HasPrefix(ct, "multipart/form-data") {
			logger.Info("request",
				zap.String("requestId", reqID),
				zap.String("method", c.Method()),
				zap.String("path", c.OriginalURL()),
				zap.String("contentType", c.Get("Content-Type")),
				zap.Int("contentLength", len(c.Body())), // ok as size only
			)
			return c.Next()
		}

		// JSON / others: log body (with masking)
		body := c.Body()
		maskedBody := string(body)

		// only mask for JSON-like
		if strings.Contains(ct, "application/json") {
			maskedBody = rePassword.ReplaceAllString(maskedBody, `"password":"****"`)
		}

		logger.Info("request",
			zap.String("requestId", reqID),
			zap.String("method", c.Method()),
			zap.String("path", c.OriginalURL()),
			zap.String("contentType", c.Get("Content-Type")),
			zap.String("body", maskedBody),
		)

		return c.Next()
	}
}
