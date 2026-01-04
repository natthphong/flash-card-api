package daily_plans

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
	"strconv"
)

func NewDailyPlansInquiryHandler(
	dailyPlansInquiryFunc DailyPlansInquiryFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		logger := logz.NewLogger()
		requestId := c.Get("requestId")
		userId := c.Locals("userId").(string)
		id, err := strconv.Atoi(userId)
		if err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.InternalError(c, api.SomeThingWentWrong)
		}
		res, err := dailyPlansInquiryFunc(ctx, logger, id)
		if err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.InternalError(c, err.Error())
		}
		return api.Ok(c, res)
	}
}
