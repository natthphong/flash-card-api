package flashcard_sets

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

func NewDeleteHandler(
	deleteFlashCardSetsFunc DeleteFlashCardSetsFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req FlashCardSetsDeleteRequest

		ctx := c.Context()
		logger := logz.NewLogger()
		requestId := c.Get("requestId")

		if err := c.BodyParser(&req); err != nil {
			logger.Error("body parse error", zap.String("requestId", requestId), zap.Error(err))
			return api.BadRequest(c, api.InvalidateBody)
		}
		if err := req.Validate(); err != nil {
			logger.Error("validation error", zap.String("requestId", requestId), zap.Error(err))
			return api.BadRequest(c, err.Error())
		}

		req.UserId = c.Locals("userId").(string)

		if err := deleteFlashCardSetsFunc(ctx, logger, req); err != nil {
			logger.Error("delete failed", zap.String("requestId", requestId), zap.Error(err))
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		return api.Ok(c, nil)
	}
}
