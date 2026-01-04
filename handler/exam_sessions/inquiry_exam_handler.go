package exam_sessions

import (
	"errors"
	"github.com/gofiber/fiber/v2"
	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
	"strconv"
)

func NewInquiryExamHandler(
	getSession GetExamSessionFunc,
	getDetails GetFlashCardDetailsFromIdsFunc,
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

		sess, err := getSession(ctx, logger, examID)
		if err != nil {
			if errors.Is(err, errors.New("exam session not found")) {
				return api.NotFoundError(c, "exam session not found")
			}
			logger.Error("get session failed", zap.Error(err))
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		totalQuestions := len(sess.QuestionIDs)
		questions, err := getDetails(ctx, logger, sess.QuestionIDs)
		if err != nil {
			return api.InternalError(c, api.SomeThingWentWrong)
		}

		resp := InquiryExamResponse{
			Id:             decimal.NewFromInt(int64(examID)),
			Questions:      questions,
			IsSubmitted:    sess.IsSubmitted,
			Score:          sess.Score,
			OwnerID:        sess.OwnerID,
			OwnerName:      sess.OwnerName,
			ExpireAt:       sess.ExpiresAt,
			TotalQuestions: totalQuestions,
			Answers:        sess.Answers,
		}
		return api.Ok(c, resp)
	}
}
