package middleware

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"

	"github.com/gofiber/fiber/v2"
)

var ignorePaths = []string{
	"/auth/",
	"/health",
	"/admin/",
	"/job/",
}

func JWTMiddleware(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := c.Get("Authorization")

		for _, subPath := range ignorePaths {
			if strings.Contains(c.Path(), subPath) {
				return c.Next()
			}
		}
		if len(tokenString) == 0 {
			return api.JwtError(c, "Token Not Found")
		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(jwtSecret), nil
		})

		if err != nil {
			return api.JwtError(c, "Token Expired")
		}
		if !token.Valid {
			return api.JwtError(c, "Invalid or expired token")
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return api.JwtError(c, "Failed to parse token claims")
		}
		id, ok := claims["userId"].(string)
		if !ok {
			return api.Forbidden(c)
		}
		if !ok {
			return api.Forbidden(c)
		}
		c.Locals("userId", id)
		return c.Next()
	}
}
