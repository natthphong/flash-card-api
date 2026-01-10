package learn

import (
	"context"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
	"go.uber.org/zap"
)

func NewReviewSubmitHandler(
	subMitReviewFunc SubMitReviewFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req ReviewSubMitRequest
		ctx := c.Context()
		logger := logz.NewLogger()
		requestId := c.Get("requestId")
		logger.Info("start", zap.String("requestId", requestId))
		err := c.BodyParser(&req)
		if err != nil {
			logger.Warn("failed to parse request", zap.Error(err))
			return api.InternalError(c, "invalid request")
		}
		validate := validator.New()
		err = validate.Struct(req)
		if err := validate.Struct(req); err != nil {
			return api.ValidationErrorResponse(c, err, req)
		}
		err = subMitReviewFunc(ctx, logger, InsertReviewLogDto{
			UserIdToken:  utils.GetUserIDToken(c),
			CardId:       req.CardId,
			Source:       req.Source,
			IsCorrect:    req.IsCorrect,
			Grade:        req.Grade,
			Streak:       0,
			Box:          1,
			AnswerDetail: req.AnswerDetail,
			NextReview:   time.Now(),
		})
		if err != nil {
			return err
		}
		return api.Ok(c, nil)
	}
}

type SubMitReviewFunc func(ctx context.Context, logger *zap.Logger, insertReviewLogDto InsertReviewLogDto) error

func NewSubMitReviewFunc(
	db *pgxpool.Pool,
	insertReviewLogFunc InsertReviewLogFunc,
	insertAndMergeUserFlashCardSrsFunc InsertAndMergeUserFlashCardSrsFunc,
) SubMitReviewFunc {
	return func(ctx context.Context, logger *zap.Logger, insertReviewLogDto InsertReviewLogDto) error {
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
		err = insertReviewLogFunc(ctx, logger, tx, insertReviewLogDto)
		if err != nil {
			logger.Error("tx commit failed", zap.Error(err))
			err = errors.New(api.SomeThingWentWrong)
		}
		err = insertAndMergeUserFlashCardSrsFunc(ctx, logger, tx, insertReviewLogDto)
		if err != nil {
			logger.Error("tx commit failed", zap.Error(err))
			err = errors.New(api.SomeThingWentWrong)
		}

		return nil
	}
}
