package job

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"go.uber.org/zap"
)

type InsertDailyPlansFunc func(ctx context.Context, logger *zap.Logger) error

func NewInsertDailyPlansFunc(db *pgxpool.Pool) InsertDailyPlansFunc {
	return func(ctx context.Context, logger *zap.Logger) error {
		var (
			currentPlanDate int64
		)
		const sqlCheck = `
			select count(*)
			from tbl_daily_plans
			where plan_date = now()::date
		`

		err := db.QueryRow(ctx, sqlCheck).Scan(&currentPlanDate)
		if err != nil {
			logger.Error(err.Error())
			return errors.New("failed to check daily plans")
		}

		if currentPlanDate > 0 {
			return errors.New("current plan date have already")
		}

		const sqlInsertPlanDate = `
				INSERT INTO tbl_daily_plans
				  (user_id, plan_date, card_ids, create_at, create_by)
				SELECT
				  u.id        AS user_id,
				  now()::date AS plan_date,
				  a.card_ids,
				  now(),
				  'SYSTEM'
				FROM tbl_users u
				CROSS JOIN LATERAL (
				  SELECT
					array_agg(x.id) AS card_ids
				  FROM (
					SELECT
					  f.id,
					  (f.status = 'studying') AS is_studying,
					  f.update_at
					FROM tbl_study_class_members m
					JOIN tbl_study_class_sets    cs ON m.class_id = cs.class_id
					JOIN tbl_flashcards          f  ON cs.set_id    = f.set_id
					WHERE m.user_id   = u.id
					  AND f.is_deleted = 'N'
					UNION ALL
					SELECT
					  f2.id,
					  (f2.status = 'studying') AS is_studying,
					  f2.update_at
					FROM tbl_flashcard_sets fs
					JOIN tbl_flashcards         f2 ON fs.id        = f2.set_id
					WHERE fs.owner_id   = u.id
					  AND fs.is_deleted = 'N'
					  AND f2.is_deleted = 'N'
					ORDER BY is_studying DESC, update_at, id
					LIMIT u.daily_target
				  ) AS x
				) AS a
				WHERE u.daily_active = 'Y'
				ORDER BY u.id
			`
		_, err = db.Exec(ctx, sqlInsertPlanDate)
		if err != nil {
			logger.Error(err.Error())
			return errors.New(api.SomeThingWentWrong)
		}

		return nil
	}
}
