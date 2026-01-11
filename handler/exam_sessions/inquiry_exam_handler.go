package exam_sessions

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

func NewInquiryExamHandler(
	getSession GetExamSessionFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		examIdStr := c.Params("examId")
		if examIdStr == "" {
			return api.BadRequest(c, "examId is required")
		}
		examID, err := strconv.Atoi(examIdStr)
		if err != nil {
			logger := logz.NewLogger()
			logger.Error("invalid examId", zap.String("requestId", c.Get("requestId")), zap.Error(err))
			return api.BadRequest(c, "examId must be a number")
		}

		ctx := c.Context()
		logger := logz.NewLogger()

		resp, err := getSession(ctx, logger, int64(examID))
		if err != nil {
			if errors.Is(err, errors.New("exam session not found")) {
				return api.NotFoundError(c, "exam session not found")
			}
			logger.Error("get session failed", zap.Error(err))
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		return api.Ok(c, resp)
	}
}
