package job

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

func NewDailyPlansCronHandler(
	insertDailyPlansFunc InsertDailyPlansFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		logger := logz.NewLogger()
		requestId := c.Get("requestId")
		err := insertDailyPlansFunc(ctx, logger)
		if err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.InternalError(c, err.Error())
		}
		return api.Ok(c, nil)
	}
}
