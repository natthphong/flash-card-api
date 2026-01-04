package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"time"
)

func RefreshTokenHandler(
	db *pgxpool.Pool,
	jwtSecret string,
	accessTokenDuration,
	refreshTokenDuration time.Duration,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req RefreshTokenRequest
		if err := c.BodyParser(&req); err != nil {
			return api.BadRequest(c, api.InvalidateBody)
		}

		token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			return api.Unauthorized(c)
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || claims["username"] == nil {
			return api.Unauthorized(c)
		}

		username, ok1 := claims["username"].(string)
		if !ok1 {
			return api.Unauthorized(c)
		}

		response, err := GenerateJWTForUser(c.Context(), db, username, "", jwtSecret, accessTokenDuration, refreshTokenDuration, true)
		if err != nil {
			return err
		}

		return api.Ok(c, response)
	}
}
