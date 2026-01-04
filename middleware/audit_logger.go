package middleware

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
	"regexp"
)

func AuditLogger() fiber.Handler {
	return func(c *fiber.Ctx) error {
		logger := logz.NewLogger()
		body := c.Body()
		reqID := c.Get("requestId")
		re := regexp.MustCompile(`(?i)"password"\s*:\s*"(.*?)"`)
		maskedBody := re.ReplaceAllString(string(body), `"password":"****"`)

		logger.Info("request", zap.String("requestId", reqID), zap.String("body", maskedBody))

		err := c.Next()
		//resBody := c.Response().Body()
		//logger.Info("response", zap.String("requestId", reqID), zap.String("body", string(resBody)))
		return err
	}
}
