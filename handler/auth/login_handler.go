package auth

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"time"
)

func LoginHandler(db *pgxpool.Pool, jwtSecret string, accessTokenDuration, refreshTokenDuration time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req LoginRequest
		ctx := c.Context()
		if err := c.BodyParser(&req); err != nil {
			return api.BadRequest(c, "Invalid input")
		}

		response, err := GenerateJWTForUser(ctx, db, req.Username, req.Password, jwtSecret, accessTokenDuration, refreshTokenDuration, false)
		if err != nil {
			return api.InternalError(c, err.Error())
		}

		return api.Ok(c, response)
	}
}
