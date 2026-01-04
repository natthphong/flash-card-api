package auth

import (
	"github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/config"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/httputil"
)

func GetRouter(group fiber.Router,
	config config.Config,
	redisCMD *redis.UniversalClient,
	dbPool *pgxpool.Pool,
	postFunc httputil.HTTPPostRequestFunc,
) {
	JwtConfig := config.JwtAuthConfig
	group.Get("/me", MeHandler(JwtConfig.JwtSecret))
	authGroup := group.Group("/auth")
	authGroup.Post("/register", NewRegisterHandler(dbPool, postFunc))
	authGroup.Post("/login", LoginHandler(
		dbPool,
		JwtConfig.JwtSecret,
		JwtConfig.AccessTokenDuration,
		JwtConfig.RefreshTokenDuration,
	))
	authGroup.Post("/refreshToken", RefreshTokenHandler(
		dbPool,
		JwtConfig.JwtSecret,
		JwtConfig.AccessTokenDuration,
		JwtConfig.RefreshTokenDuration,
	))
}
