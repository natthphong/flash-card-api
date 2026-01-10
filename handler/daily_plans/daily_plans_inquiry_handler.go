package daily_plans

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
	"go.uber.org/zap"
)

func NewDailyPlansInquiryHandler(
	dailyPlansInquiryFunc DailyPlansInquiryFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		logger := logz.NewLogger()
		requestId := c.Get("requestId")
		userIdToken := utils.GetUserIDToken(c)
		res, err := dailyPlansInquiryFunc(ctx, logger, userIdToken)
		if err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.InternalError(c, err.Error())
		}
		return api.Ok(c, res)
	}
}
