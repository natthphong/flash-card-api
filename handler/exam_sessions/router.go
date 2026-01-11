package exam_sessions

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
	examGroup := group.Group("/exam-sessions")
	examGroup.Post("", NewExamSessionsListHandler(
		NewListExamHistory(dbPool),
	))
	examGroup.Post("/start", NewStartExamHandler(
		NewSelectQuestionIds(dbPool),
		NewInsertStartExamSessions(dbPool),
		NewGetFlashCardDetailsFromIds(dbPool),
	))

	examGroup.Get("/:examId", NewInquiryExamHandler(
		NewGetExamSession(dbPool),
	))

	examGroup.Put("/answer", NewUpdateExamHandler(
		NewUpdateExamSession(dbPool),
	))
	examGroup.Put("/submit/:examId", NewSubmitHandler(
		NewGetExamSession(dbPool),
		NewSubMitReviewFunc(
			dbPool,
			NewInsertReviewLogsFunc(),
			NewUpsertUserFlashcardSrsBatchFunc(),
		),
	))
}
