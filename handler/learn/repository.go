package learn

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type InsertReviewLogDto struct {
	UserIdToken  string `json:"userIdToken"`
	CardId       decimal.Decimal
	Source       string
	IsCorrect    string
	Grade        int
	Streak       int
	Box          int
	AnswerDetail json.RawMessage
	NextReview   time.Time
}

type InsertReviewLogFunc func(ctx context.Context, logger *zap.Logger, tx pgx.Tx, insertReviewLogDto InsertReviewLogDto) error

func NewInsertReviewLogFunc() InsertReviewLogFunc {
	return func(ctx context.Context, logger *zap.Logger, tx pgx.Tx, dto InsertReviewLogDto) error {

		sql := `	
		insert into tbl_review_log (user_id_token, card_id, source, grade, is_correct, answer_detail)
		values ($1,$2,$3,$4,$5,$6);
		`
		_, err := tx.Exec(ctx, sql, dto.UserIdToken, dto.CardId, dto.Source, dto.Grade, dto.IsCorrect, dto.AnswerDetail)
		if err != nil {
			return err
		}

		return nil
	}
}

type InsertAndMergeUserFlashCardSrsFunc func(ctx context.Context, logger *zap.Logger, tx pgx.Tx, dto InsertReviewLogDto) error

func NewInsertAndMergeUserFlashCardSrsFunc() InsertAndMergeUserFlashCardSrsFunc {
	return func(ctx context.Context, logger *zap.Logger, tx pgx.Tx, dto InsertReviewLogDto) error {
		const sql = `
		WITH input AS (
		  SELECT
			$1::varchar(36) AS user_id_token,
			$2::bigint      AS card_id,
			$3::smallint    AS grade
		),
		upsert AS (
		  INSERT INTO tbl_user_flashcard_srs(
			user_id_token, card_id,
			box, next_review_at, last_review_at,
			streak, total_reviews, last_grade,
			created_at, updated_at
		  )
		  SELECT
			i.user_id_token,
			i.card_id,
			CASE
			  WHEN i.grade <= 2 THEN 1
			  WHEN i.grade = 3 THEN 1
			  ELSE 2
			END AS box,
			CASE
			  WHEN i.grade <= 2 THEN now() + make_interval(days => 1)
			  WHEN i.grade = 3 THEN now() + make_interval(days => 1)
			  ELSE now() + make_interval(days => 2)
			END AS next_review_at,
			now() AS last_review_at,
			CASE
			  WHEN i.grade <= 2 THEN 0
			  ELSE 1
			END AS streak,
			1 AS total_reviews,
			i.grade AS last_grade,
			now() AS created_at,
			now() AS updated_at
		  FROM input i
		  ON CONFLICT (user_id_token, card_id)
		  DO UPDATE SET
			box = (
			  CASE
				WHEN EXCLUDED.last_grade <= 2 THEN 1
				WHEN EXCLUDED.last_grade = 3 THEN tbl_user_flashcard_srs.box
				ELSE LEAST(tbl_user_flashcard_srs.box + 1, 5)
			  END
			),
			streak = (
			  CASE
				WHEN EXCLUDED.last_grade <= 2 THEN 0
				ELSE tbl_user_flashcard_srs.streak + 1
			  END
			),
			next_review_at = (
			  CASE
				WHEN EXCLUDED.last_grade <= 2 THEN now() + make_interval(days => 1)
				ELSE
				  now() + make_interval(days =>
					CASE
					  WHEN (CASE
						WHEN EXCLUDED.last_grade <= 2 THEN 1
						WHEN EXCLUDED.last_grade = 3 THEN tbl_user_flashcard_srs.box
						ELSE LEAST(tbl_user_flashcard_srs.box + 1, 5)
					  END) = 1 THEN 1
					  WHEN (CASE
						WHEN EXCLUDED.last_grade <= 2 THEN 1
						WHEN EXCLUDED.last_grade = 3 THEN tbl_user_flashcard_srs.box
						ELSE LEAST(tbl_user_flashcard_srs.box + 1, 5)
					  END) = 2 THEN 2
					  WHEN (CASE
						WHEN EXCLUDED.last_grade <= 2 THEN 1
						WHEN EXCLUDED.last_grade = 3 THEN tbl_user_flashcard_srs.box
						ELSE LEAST(tbl_user_flashcard_srs.box + 1, 5)
					  END) = 3 THEN 4
					  WHEN (CASE
						WHEN EXCLUDED.last_grade <= 2 THEN 1
						WHEN EXCLUDED.last_grade = 3 THEN tbl_user_flashcard_srs.box
						ELSE LEAST(tbl_user_flashcard_srs.box + 1, 5)
					  END) = 4 THEN 8
					  ELSE 16
					END
				  )
			  END
			),
			last_review_at = now(),
			total_reviews = tbl_user_flashcard_srs.total_reviews + 1,
			last_grade = EXCLUDED.last_grade,
			updated_at = now()
		  RETURNING 1
		)
		SELECT 1 FROM upsert;
		`
		_, err := tx.Exec(ctx, sql, dto.UserIdToken, dto.CardId, dto.Grade)
		return err
	}
}
