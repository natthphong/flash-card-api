package exam_sessions

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
	"go.uber.org/zap"
)

func NewStartExamHandler(
	selectQuestionIds SelectQuestionIdsFunc,
	insertSession InsertStartExamSessionsFunc,
	getDetails GetFlashCardDetailsFromIdsFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req StartExamRequest
		var questionIDs []int64
		var err error
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
		if req.TimeLimitSeconds != nil && *req.TimeLimitSeconds > 0 {
			t := time.Now().Add(time.Duration(*req.TimeLimitSeconds) * time.Second)
			req.TimeLimit = &t
		}
		req.UserId = utils.GetUserID(c)
		userId := utils.GetUserIDToken(c)
		if req.SetId != nil && !req.SetId.IsZero() {
			questionIDs, err = selectQuestionIds(ctx, logger, req)
			if err != nil || len(questionIDs) == 0 {
				logger.Error("no questions ", zap.String("requestId", requestId), zap.Error(err))
				return api.BadRequest(c, "Questions not found")
			}
		}
		if req.DailyPlanId != nil && !req.DailyPlanId.IsZero() {
			//TODO
		}

		// insert session
		sessionId, err := insertSession(ctx, logger, userId, req, questionIDs)
		if err != nil {
			return api.InternalError(c, api.SomeThingWentWrong)
		}
		// TODO to remove it
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
