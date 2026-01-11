package exam_sessions

import (
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

func NewUpdateExamHandler(
	updateExamSessionFunc UpdateExamSessionFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var req ExamSessionUpdateRequest

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

		correct, err := updateExamSessionFunc(ctx, logger, req)
		if err != nil {
			return api.InternalError(c, err.Error())
		}

		return api.Ok(c, fiber.Map{
			"correct": correct,
		})
	}
}
