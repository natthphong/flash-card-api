package exam_sessions

import (
	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
	"strconv"
)

func NewStartExamHandler(
	selectQuestionIds SelectQuestionIdsFunc,
	insertSession InsertStartExamSessionsFunc,
	getDetails GetFlashCardDetailsFromIdsFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req StartExamRequest
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
		uidStr := c.Locals("userId").(string)
		userId, err := strconv.Atoi(uidStr)
		if err != nil {
			logger.Error("atoi userId", zap.String("requestId", requestId), zap.Error(err))
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		questionIDs, err := selectQuestionIds(ctx, logger, req)
		if err != nil || len(questionIDs) == 0 {
			logger.Error("no questions ", zap.String("requestId", requestId), zap.Error(err))
			return api.BadRequest(c, "Questions not found")
		}

		// insert session
		sessionId, err := insertSession(ctx, logger, userId, questionIDs, req.TimeLimit, req.Username)
		if err != nil {
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		// fetch details
		questions, err := getDetails(ctx, logger, questionIDs)
		if err != nil {
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		// respond
		resp := StartExamResponse{
			Id:          decimal.NewFromInt(sessionId),
			Questions:   questions,
			IsSubmitted: "N",
			ExpireAt:    req.TimeLimit,
		}
		return api.Ok(c, resp)
	}
}
