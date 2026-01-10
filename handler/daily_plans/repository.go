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

type UserConfigDto struct {
	DailyActive       string          `json:"dailyActive"`
	DailyTarget       decimal.Decimal `json:"dailyTarget"`
	DailySetId        decimal.Decimal `json:"dailySetId"`
	DefaultSetId      decimal.Decimal `json:"defaultSetId"`
	DailySetIdTitle   string          `json:"dailySetIdTitle"`
	DailySetIdDesc    string          `json:"dailySetIdDesc"`
	DefaultSetIdTitle string          `json:"defaultSetIdTitle"`
	DefaultSetIdDesc  string          `json:"defaultSetIdDesc"`
	CreateDateTime    time.Time       `json:"createDateTime"`
}

type UpdateDailyPlansFunc func(ctx context.Context, logger *zap.Logger, req DailyPlanSettingRequest) error

func NewUpdateDailyPlansFunc(db *pgxpool.Pool) UpdateDailyPlansFunc {
	return func(ctx context.Context, logger *zap.Logger, req DailyPlanSettingRequest) (err error) {
		const sql = `
            UPDATE tbl_user_config
               SET daily_active = COALESCE($1, daily_active),
                   daily_target = COALESCE($2, daily_target),
               	   daily_flash_card_set_id = COALESCE($3, daily_flash_card_set_id),
               	   default_flash_card_set_id = COALESCE($4, default_flash_card_set_id),
               		update_at = now()
             WHERE user_id_token = $5
        `
		_, err = db.Exec(ctx, sql, req.DailyActive, req.DailyTarget, req.DailySetId, req.DefaultSetId, req.UserIdToken)
		if err != nil {
			logger.Error("failed to update daily plans", zap.Error(err))
			return errors.New(api.SomeThingWentWrong)
		}
		return err
	}
}

type GetConfigDailyPlanFunc func(ctx context.Context, logger *zap.Logger, userIdToken string) (UserConfigDto, error)

func NewGetConfigDailyPlanFunc(db *pgxpool.Pool) GetConfigDailyPlanFunc {
	return func(ctx context.Context, logger *zap.Logger, userIdToken string) (UserConfigDto, error) {
		userConfigDto := UserConfigDto{}
		sql := `
		SELECT
			tuc.daily_active,
			tuc.daily_target,
			tuc.daily_flash_card_set_id,
			tuc.default_flash_card_set_id,
			tuc.create_at,
		
			dfs.title   AS daily_flash_card_set_title,
			dfs.description AS daily_flash_card_set_description,
		
			dfts.title  AS default_flash_card_set_title,
			dfts.description AS default_flash_card_set_description
		
		FROM tbl_user_config tuc
		LEFT JOIN tbl_flashcard_sets dfs
			   ON tuc.daily_flash_card_set_id = dfs.id
		LEFT JOIN tbl_flashcard_sets dfts
			   ON tuc.default_flash_card_set_id = dfts.id
		WHERE tuc.user_id_token = $1;

		`
		err := db.QueryRow(ctx, sql, userIdToken).Scan(&userConfigDto.DailyActive, &userConfigDto.DailyTarget,
			&userConfigDto.DailySetId, &userConfigDto.DefaultSetId, &userConfigDto.CreateDateTime,
			&userConfigDto.DailySetIdTitle, &userConfigDto.DailySetIdDesc,
			&userConfigDto.DefaultSetIdTitle, &userConfigDto.DefaultSetIdDesc,
		)
		if err != nil {
			return UserConfigDto{}, errors.New(api.NotFound)
		}
		return userConfigDto, nil
	}
}

type DailyPlansInquiryFunc func(
	ctx context.Context,
	logger *zap.Logger,
	userId string,
) ([]DailyFlashCardSetsInquiryResponse, error)

func NewDailyPlansInquiry(db *pgxpool.Pool) DailyPlansInquiryFunc {
	return func(
		ctx context.Context,
		logger *zap.Logger,
		userId string,
	) ([]DailyFlashCardSetsInquiryResponse, error) {
		const sql = `
            WITH plan AS (
                SELECT card_ids, id
                  FROM tbl_daily_plans
                 WHERE user_id_token    = $1
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
			logger.Error("failed to query daily plan cards", zap.Error(err), zap.String("user_id_token", userId))
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
				logger.Error("scan daily plan card failed", zap.Error(err), zap.String("user_id_token", userId))
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
			logger.Error("iterating daily plan card rows failed", zap.Error(err), zap.String("user_id_token", userId))
			return nil, errors.New(api.SomeThingWentWrong)
		}

		return result, nil
	}
}
