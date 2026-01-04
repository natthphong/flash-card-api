package admin

import (
	"github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/config"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/httputil"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/middleware"
)

func GetRouter(group fiber.Router,
	config config.Config,
	redisCMD *redis.UniversalClient,
	dbPool *pgxpool.Pool,
	postFunc httputil.HTTPPostRequestFunc,
) {
	adminGroup := group.Group("/admin")
	adminGroup.Use(middleware.JWTMiddlewareAdmin())
	adminGroup.Post("/user/confirm", NewConfirmUserHandler(dbPool))

}
