package utils

import (
	"strings"

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
func GetIndexFromString(str string) int {
	str = strings.ToUpper(strings.TrimSpace(str))
	if len(str) == 0 {
		return 0
	}
	c := str[0]
	if c < 'A' || c > 'Z' {
		return 0
	}
	return int(c-'A') + 1
}
