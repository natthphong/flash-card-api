package daily_plans

import (
	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

func NewDailyPlanSettingHandler(
	updateDailyPlansFunc UpdateDailyPlansFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var (
			req DailyPlanSettingRequest
			err error
		)
		ctx := c.Context()
		logger := logz.NewLogger()
		requestId := c.Get("requestId")
		if err := c.BodyParser(&req); err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.BadRequest(c, api.InvalidateBody)
		}
		if err := req.Validate(); err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.BadRequest(c, err.Error())
		}
		userId := c.Locals("userId").(string)
		req.UserId, err = decimal.NewFromString(userId)
		if err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.InternalError(c, api.SomeThingWentWrong)
		}
		err = updateDailyPlansFunc(ctx, logger, req)
		if err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.InternalError(c, err.Error())
		}

		return api.Ok(c, nil)
	}
}
