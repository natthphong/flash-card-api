package utils

import (
	"github.com/gofiber/fiber/v2"
)

const (
	PRACTICE = "PRACTICE"
	EXAM     = "EXAM"
	DAILY    = "DAILY"
)

func GetUserIDToken(c *fiber.Ctx) string {
	userIdToken := c.Locals("userIdToken").(string)
	return userIdToken
}

func GetUserID(c *fiber.Ctx) string {
	userId := c.Locals("userId").(string)
	return userId
}
