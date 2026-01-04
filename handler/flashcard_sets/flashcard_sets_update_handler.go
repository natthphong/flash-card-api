package flashcard_sets

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
)

func NewUpdateHandler(
	updateFlashCardSetsFunc UpdateFlashCardSetsFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req FlashCardSetsUpdateRequest

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

		req.Username = c.Locals("username").(string)

		if err := updateFlashCardSetsFunc(ctx, logger, req); err != nil {
			logger.Error("update failed", zap.String("requestId", requestId), zap.Error(err))
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		return api.Ok(c, nil)
	}
}
