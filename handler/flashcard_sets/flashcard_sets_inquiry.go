package flashcard_sets

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

func NewInquiryFlashCardSetsHandler(
	inquiryFlashCardSetsFunc FlashCardSetsInquiryFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		setId := c.Params("setId")

		ctx := c.Context()
		logger := logz.NewLogger()
		requestId := c.Get("requestId")
		if setId == "" {
			return api.BadRequest(c, "setId is required")
		}
		id, err := strconv.Atoi(setId)
		if err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		userIdStr := c.Locals("userId").(string)
		userId, err := strconv.Atoi(userIdStr)
		if err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		res, err := inquiryFlashCardSetsFunc(ctx, logger, id, userId)
		if err != nil {
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		return api.Ok(c, res)
	}
}
