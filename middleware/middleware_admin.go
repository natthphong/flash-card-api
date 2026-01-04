package middleware

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
)

func JWTMiddlewareAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := c.Get("X-auth-token")

		if len(tokenString) == 0 {
			return api.JwtError(c, "Token Not Found")
		}
		// TODO
		if tokenString != "V1ZkU2RHRlhOVlZpTW5Sc1lteGFiR051YkZSa1NFcDJZbTFrVldJeWRHeGlaejA5" {
			return api.JwtError(c, "Invalid or expired token")
		}

		return c.Next()
	}
}
