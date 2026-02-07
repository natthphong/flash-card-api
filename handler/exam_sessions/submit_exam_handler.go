package exam_sessions

import (
	"context"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
	"go.uber.org/zap"
)

func NewSubmitHandler(
	getSessionFunc GetExamSessionFunc,
	examSubMitReviewFunc ExamSubMitReviewFunc,
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

		// Validate
		//requestId := c.Get("requestId")
		sessionDto, err := getSessionFunc(ctx, logger, int64(examID))
		if err != nil {
			return err
		}
		if sessionDto.Status == EXPIRED {
			return api.BadRequest(c, "session is expired")
		}
		examItems := []ExamSubmitItem{}
		// TODO want new logic for cal grade
		totalScore := 0
		for _, item := range sessionDto.Questions {
			if item.Answer == nil || item.Answer.IsCorrect == nil {
				return api.BadRequest(c, "this all questions must have answer")
			}
			isCorrect := *item.Answer.IsCorrect
			grade := 0
			if isCorrect == utils.FlagY {
				totalScore++
				grade = 4
			}
			examItems = append(examItems, ExamSubmitItem{
				CardID:       item.CardID,
				Source:       utils.EXAM,
				IsCorrect:    isCorrect,
				Grade:        int16(grade),
				AnswerDetail: item.Answer.Detail,
			})
		}
		userIdToken := utils.GetUserIDToken(c)
		err = examSubMitReviewFunc(ctx, logger, userIdToken, examItems, sessionDto.ID, totalScore)
		if err != nil {
			logger.Error(err.Error())
			return api.InternalError(c, err.Error())
		}

		return api.Ok(c, fiber.Map{
			"score": totalScore,
		})
	}
}

type ExamSubMitReviewFunc func(ctx context.Context, logger *zap.Logger, userIdToken string, items []ExamSubmitItem, examSessionId int64, totalScore int) error

func NewSubMitReviewFunc(
	db *pgxpool.Pool,
	insertReviewLogFunc InsertReviewLogsFunc,
	insertAndMergeUserFlashCardSrsFunc UpsertUserFlashcardSrsBatchFunc,
	updateExamSessionAfterSubmitFunc UpdateExamSessionAfterSubmitFunc,
) ExamSubMitReviewFunc {
	return func(ctx context.Context, logger *zap.Logger, userIdToken string, items []ExamSubmitItem, examSessionId int64, totalScore int) error {
		tx, err := db.Begin(ctx)
		if err != nil {
			logger.Error("failed to begin tx", zap.Error(err))
			return errors.New(api.SomeThingWentWrong)
		}
		defer func() {
			if err != nil {
				if rbErr := tx.Rollback(ctx); rbErr != nil {
					logger.Error("tx rollback failed", zap.Error(rbErr))
				}
			} else if cmErr := tx.Commit(ctx); cmErr != nil {
				logger.Error("tx commit failed", zap.Error(cmErr))
				err = errors.New(api.SomeThingWentWrong)
			}
		}()

		err = insertReviewLogFunc(ctx, logger, tx, userIdToken, items)
		if err != nil {
			logger.Error("tx commit failed", zap.Error(err))
			err = errors.New(api.SomeThingWentWrong)
		}
		err = insertAndMergeUserFlashCardSrsFunc(ctx, logger, tx, userIdToken, items)
		if err != nil {
			logger.Error("tx commit failed", zap.Error(err))
			err = errors.New(api.SomeThingWentWrong)
		}
		err = updateExamSessionAfterSubmitFunc(ctx, logger, tx, examSessionId, totalScore)
		if err != nil {
			logger.Error("tx commit failed", zap.Error(err))
			err = errors.New(api.SomeThingWentWrong)
		}
		return nil
	}
}
