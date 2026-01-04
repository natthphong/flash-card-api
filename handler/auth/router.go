package auth

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/adapter"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/config"
)

func GetRouter(group fiber.Router,
	config config.Config,
	adapterHomeServer adapter.Adapter,
) {
	JwtConfig := config.JwtAuthConfig
	group.Get("/me", MeHandler(JwtConfig.JwtSecret))
	authGroup := group.Group("/auth")
	//authGroup.Post("/register", NewRegisterHandler(dbPool, postFunc))
	authGroup.Post("/login", LoginHandler(
		adapterHomeServer,
		&config,
	))

}
