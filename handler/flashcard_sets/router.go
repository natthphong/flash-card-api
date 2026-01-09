package flashcard_sets

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
	flashCardSetsGroup := group.Group("/flashcard-sets")

	flashCardSetsGroup.Post("/create", NewCreateHandler(
		NewInsertFlashCardsSet(dbPool),
	))
	flashCardSetsGroup.Post("/import/csv", NewFlashCardSetsImportCsvHandler(
		NewInsertFlashCardsSet(dbPool),
	))

	flashCardSetsGroup.Put("/update", NewUpdateHandler(
		NewUpdateFlashCardSets(dbPool),
	))
	flashCardSetsGroup.Post("/delete", NewDeleteHandler(
		NewDeleteFlashCardSets(dbPool),
	))
	flashCardSetsGroup.Post("/list", NewListHandler(
		NewListFlashCardSets(dbPool),
	))

	flashCardSetsGroup.Post("/duplicate", NewDuplicateFlashCardsSetHandler(
		NewDuplicateFlashCardsSet(dbPool),
	))
	// enhance
	flashCardSetsGroup.Post("/track", NewInsertAndMergeFlashCardSetsTrackerHandler(
		NewInsertAndMergeFlashCardSetsTracker(dbPool),
	))
	// enhance
	flashCardSetsGroup.Post("/reset", NewResetStatusHandler(
		NewResetStatusFlashCards(dbPool),
	))

	flashCardSetsGroup.Get("/:setId", NewInquiryFlashCardSetsHandler(
		NewFlashCardSetsInquiry(dbPool),
	))

	flashCards := group.Group("/flashcards")

	flashCards.Post("/create", NewFlashCardCreateHandler(
		NewInsertFlashCards(dbPool),
	))
	flashCards.Post("/delete", NewFlashCardsDeleteHandler(
		NewDeleteFlashCards(dbPool),
	))

	flashCards.Put("/update", NewFlashCardsUpdateHandler(
		NewUpdateFlashCards(dbPool),
	))

}
