package flashcard_sets

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

func NewFlashCardsUpdateHandler(
	updateFlashCardsFunc UpdateFlashCardsFunc,
) fiber.Handler {

	return func(c *fiber.Ctx) error {
		var (
			req FlashCardsUpdateRequest
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
		err := updateFlashCardsFunc(ctx, logger, req)
		if err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.InternalError(c, api.SomeThingWentWrong)
		}
		username := c.Locals("username").(string)
		req.Username = username

		return api.Ok(c, nil)
	}
}
