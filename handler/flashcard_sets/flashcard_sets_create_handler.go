package flashcard_sets

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

func NewCreateHandler(
	insertFlashCardsSetFunc InsertFlashCardsSetFunc,
) fiber.Handler {

	return func(c *fiber.Ctx) error {

		var (
			req FlashCardSetsCreateRequest
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
		userIdToken := c.Locals("userIdToken").(string)
		userId := c.Locals("userId").(string)

		req.UserId = userId
		req.OwnerIdToken = userIdToken
		err := insertFlashCardsSetFunc(ctx, logger, req)
		if err != nil {
			return api.InternalError(c, api.SomeThingWentWrong)
		}
		return api.Ok(c, nil)
	}
}
