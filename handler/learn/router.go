package learn

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetRouter(
	group fiber.Router,
	dbPool *pgxpool.Pool,

) {
	voiceGroup := group.Group("/learn")
	voiceGroup.Post("/review/submit", NewReviewSubmitHandler(
		NewSubMitReviewFunc(
			dbPool,
			NewInsertReviewLogFunc(),
			NewInsertAndMergeUserFlashCardSrsFunc(),
		),
	))
}
