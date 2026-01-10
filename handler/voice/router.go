package voice

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/adapter"
)

func GetRouter(
	group fiber.Router,
	dbPool *pgxpool.Pool,
	homeProxy adapter.Adapter,
) {
	voiceGroup := group.Group("/voice")
	voiceGroup.Post("/generate", NewVoceHandler(
		homeProxy,
		NewUpdateHitCacheAndReturnAudio(dbPool),
		NewInsertAudioUrlAndKeyToCacheFunc(dbPool),
	))
}
