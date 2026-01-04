package daily_plans

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
	dailyPlanGroup := group.Group("/daily-plans")
	dailyPlanGroup.Post("/setting", NewDailyPlanSettingHandler(
		NewUpdateDailyPlansFunc(dbPool),
	))
	dailyPlanGroup.Get("/inquiry", NewDailyPlansInquiryHandler(
		NewDailyPlansInquiry(dbPool),
	))
}
