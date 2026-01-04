package flashcard_sets

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
	"strconv"
)

func NewListHandler(
	listFlashCardSetsFunc ListFlashCardSetsFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var (
			req FlashCardSetsListRequest
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
		id, err := strconv.Atoi(userId)
		if err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		res, err := listFlashCardSetsFunc(ctx, logger, req, id)
		if err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.InternalError(c, api.SomeThingWentWrong)
		}
		return api.Ok(c, res)
	}
}
