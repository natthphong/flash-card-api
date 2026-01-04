package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"strings"
)

func MeHandler(jwtSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := c.Get("Authorization")
		if len(tokenString) == 0 {
			return api.Unauthorized(c)
		}
		tokenString = strings.TrimPrefix(tokenString, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			return api.Unauthorized(c)
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return api.Unauthorized(c)
		}
		userDetails := fiber.Map{
			"id":       claims["id"],
			"name":     claims["name"],
			"username": claims["username"],
			"email":    claims["email"],
		}
		response := map[string]interface{}{
			"jwtBody": userDetails,
		}
		return api.Ok(c, response)
	}
}
