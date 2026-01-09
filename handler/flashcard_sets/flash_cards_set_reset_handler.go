package flashcard_sets

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

func NewResetStatusHandler(
	resetFunc ResetStatusFlashCardsFunc,
) fiber.Handler {

	return func(c *fiber.Ctx) error {
		var body ResetFlashCardStatusRequest
		ctx := c.Context()
		logger := logz.NewLogger()
		requestID := c.Get("requestId")
		if err := c.BodyParser(&body); err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestID))
			return api.BadRequest(c, api.InvalidateBody)
		}
		if err := body.Validate(); err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestID))
			return api.BadRequest(c, err.Error())
		}

		if err := resetFunc(ctx, logger, ResetFlashCardStatusRequest{
			SetID: body.SetID,
		}); err != nil {
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		return api.Ok(c, nil)
	}
}
