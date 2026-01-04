package exam_sessions

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
	"go.uber.org/zap"
)

func NewUpdateExamHandler(
	getSession GetExamSessionFunc,
	getDetails GetFlashCardDetailsFromIdsFunc,
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
		req.Username = c.Locals("username").(string)
		if req.IsSubmitted != nil && *req.IsSubmitted == utils.FlagY {
			var answers AnswersDetail
			sess, err := getSession(ctx, logger, int(req.Id.IntPart()))
			if err != nil {
				if errors.Is(err, errors.New("exam session not found")) {
					return api.NotFoundError(c, "exam session not found")
				}
				logger.Error("get session failed", zap.Error(err))
				return api.InternalError(c, api.SomeThingWentWrong)
			}
			if sess.IsSubmitted == utils.FlagY {
				return api.BadRequest(c, "exam session already submitted")
			}

			if req.Answers != nil {
				answers = *req.Answers
			} else if req.Answers == nil {
				if sess.Answers == nil {
					return api.NotFoundError(c, "no answers")
				}
				if err := json.Unmarshal(*sess.Answers, &answers); err != nil {
					return api.InternalError(c, api.SomeThingWentWrong)
				}
			}

			if len(sess.QuestionIDs) != len(answers.AnswerList) {
				logger.Error("answer not complete yet",
					zap.Any("QuestionIDs", len(sess.QuestionIDs)),
					zap.Any("AnswerList", len(answers.AnswerList)),
				)
				return api.NotFoundError(c, "answer not complete yet")
			}

			// TODO check score

			questions, err := getDetails(ctx, logger, sess.QuestionIDs)
			if err != nil {
				return api.InternalError(c, api.SomeThingWentWrong)
			}
			score := 0
			for _, question := range questions {
				for _, answer := range answers.AnswerList {
					if answer.CardId.Equal(question.Id) {
						answerIgnoreAllCase := strings.ToLower(answer.Answer)
						answerIgnoreAllCase = strings.TrimSpace(answerIgnoreAllCase)
						backIgnoreAllCase := strings.ToLower(question.Back)
						backIgnoreAllCase = strings.TrimSpace(backIgnoreAllCase)
						if answerIgnoreAllCase == backIgnoreAllCase {
							score += 1
							answer.Correct = true
						}
						break
					}
				}
			}
			req.Score = &score
		}

		err := updateExamSessionFunc(ctx, logger, req)
		if err != nil {
			return api.InternalError(c, err.Error())
		}

		return api.Ok(c, req)
	}
}
