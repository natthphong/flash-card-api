package daily_plans

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"go.uber.org/zap"
)

type UpdateDailyPlansFunc func(ctx context.Context, logger *zap.Logger, req DailyPlanSettingRequest) error

func NewUpdateDailyPlansFunc(db *pgxpool.Pool) UpdateDailyPlansFunc {
	return func(ctx context.Context, logger *zap.Logger, req DailyPlanSettingRequest) (err error) {
		const sql = `
            UPDATE tbl_users
               SET daily_active = COALESCE($1, daily_active),
                   daily_target = COALESCE($2, daily_target)
             WHERE id = $3
        `
		_, err = db.Exec(ctx, sql, req.DailyActive, req.DailyTarget, req.UserId)
		if err != nil {
			logger.Error("failed to update daily plans", zap.Error(err))
			return errors.New(api.SomeThingWentWrong)
		}
		return err
	}
}

type DailyPlansInquiryFunc func(
	ctx context.Context,
	logger *zap.Logger,
	userId int,
) ([]DailyFlashCardSetsInquiryResponse, error)

func NewDailyPlansInquiry(db *pgxpool.Pool) DailyPlansInquiryFunc {
	return func(
		ctx context.Context,
		logger *zap.Logger,
		userId int,
	) ([]DailyFlashCardSetsInquiryResponse, error) {
		const sql = `
            WITH plan AS (
                SELECT card_ids, id
                  FROM tbl_daily_plans
                 WHERE user_id    = $1
                   AND plan_date  = now()::date
                   AND is_deleted = 'N'
            )
            SELECT
 				p.id as dailyPlanId,
                f.id,
                f.front,
                f.back,
                f.choices,
                f.status,
                f.create_at,
                f.create_by AS owner_name,
                u.ordinality
            FROM plan p
            CROSS JOIN LATERAL unnest(p.card_ids) WITH ORDINALITY AS u(card_id, ordinality)
            JOIN tbl_flashcards f ON f.id = u.card_id
            WHERE f.is_deleted = 'N'
            ORDER BY u.ordinality
        `

		rows, err := db.Query(ctx, sql, userId)
		if err != nil {
			logger.Error("failed to query daily plan cards", zap.Error(err), zap.Int("user_id", userId))
			return nil, errors.New(api.SomeThingWentWrong)
		}
		defer rows.Close()

		var result []DailyFlashCardSetsInquiryResponse
		for rows.Next() {
			var (
				idPlan    int64
				idInt     int64
				front     string
				back      string
				choices   []string
				status    string
				createAt  time.Time
				ownerName string
				ord       int64
			)
			if err := rows.Scan(
				&idPlan,
				&idInt,
				&front,
				&back,
				&choices,
				&status,
				&createAt,
				&ownerName,
				&ord,
			); err != nil {
				logger.Error("scan daily plan card failed", zap.Error(err), zap.Int("user_id", userId))
				return nil, errors.New(api.SomeThingWentWrong)
			}

			result = append(result, DailyFlashCardSetsInquiryResponse{
				DailyPlanId: decimal.NewFromInt(idPlan),
				Id:          decimal.NewFromInt(idInt),
				Front:       front,
				Back:        back,
				Choices:     choices,
				Status:      status,
				CreateAt:    createAt,
				OwnerName:   ownerName,
				Seq:         decimal.NewFromInt(ord),
			})
		}
		if err := rows.Err(); err != nil {
			logger.Error("iterating daily plan card rows failed", zap.Error(err), zap.Int("user_id", userId))
			return nil, errors.New(api.SomeThingWentWrong)
		}

		return result, nil
	}
}
