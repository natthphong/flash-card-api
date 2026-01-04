package exam_sessions

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"go.uber.org/zap"
	"strings"
	"time"
)

type SelectQuestionIdsFunc func(
	ctx context.Context,
	logger *zap.Logger,
	req StartExamRequest,
) ([]int64, error)

func NewSelectQuestionIds(db *pgxpool.Pool) SelectQuestionIdsFunc {
	return func(ctx context.Context, logger *zap.Logger, req StartExamRequest) ([]int64, error) {
		if req.SetId != nil {
			const sql = `
                SELECT id
                  FROM tbl_flashcards
                 WHERE set_id    = $1
                   AND is_deleted = 'N'
                order by update_at, (seq % 2),id
                limit $2
            `
			rows, err := db.Query(ctx, sql, req.SetId.IntPart(), req.ExamTotalQuestion.IntPart())
			if err != nil {
				logger.Error("query flashcards by set failed", zap.Error(err), zap.Any("setId", req.SetId))
				return nil, errors.New(api.SomeThingWentWrong)
			}
			defer rows.Close()

			var ids []int64
			for rows.Next() {
				var id int64
				if err := rows.Scan(&id); err != nil {
					return nil, err
				}
				ids = append(ids, id)
			}
			return ids, nil
		}

		if req.DailyPlanId != nil {
			const sql = `
                SELECT unnest(card_ids)
                  FROM tbl_daily_plans
                 WHERE id         = $1
                   AND is_deleted = 'N'
            `
			rows, err := db.Query(ctx, sql, req.DailyPlanId.IntPart())
			if err != nil {
				logger.Error("query daily plan card_ids failed", zap.Error(err), zap.Any("planId", req.DailyPlanId))
				return nil, errors.New(api.SomeThingWentWrong)
			}
			defer rows.Close()

			var ids []int64
			for rows.Next() {
				var id int64
				if err := rows.Scan(&id); err != nil {
					return nil, err
				}
				ids = append(ids, id)
			}
			return ids, nil
		}

		return nil, fmt.Errorf("classId-based selection not supported yet")
	}
}

type InsertStartExamSessionsFunc func(
	ctx context.Context,
	logger *zap.Logger,
	userId int,
	questionIDs []int64,
	expiresAt *time.Time,
	createBy string,
) (int64, error)

func NewInsertStartExamSessions(db *pgxpool.Pool) InsertStartExamSessionsFunc {
	return func(
		ctx context.Context,
		logger *zap.Logger,
		userId int,
		questionIDs []int64,
		expiresAt *time.Time,
		createBy string,
	) (int64, error) {
		const sql = `
		INSERT INTO tbl_exam_sessions
		  (user_id, question_ids, expires_at, create_at, create_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
		`

		var sessionId int64
		err := db.QueryRow(ctx, sql,
			userId,
			questionIDs,
			expiresAt,
			time.Now(),
			createBy,
		).Scan(&sessionId)
		if err != nil {
			logger.Error("insert exam session failed", zap.Error(err), zap.Int("userId", userId))
			return 0, errors.New(api.SomeThingWentWrong)
		}
		return sessionId, nil
	}
}

type GetFlashCardDetailsFromIdsFunc func(ctx context.Context, logger *zap.Logger, questionIDs []int64) ([]FlashCardDetails, error)

func NewGetFlashCardDetailsFromIds(db *pgxpool.Pool) GetFlashCardDetailsFromIdsFunc {
	return func(ctx context.Context, logger *zap.Logger, questionIDs []int64) ([]FlashCardDetails, error) {
		const sql = `
            SELECT
                f.id,
                f.front,
                f.back,
                f.choices,
                f.status,
                f.create_at,
                f.create_by AS owner_name,
                u.ordinality
            FROM unnest($1::bigint[]) WITH ORDINALITY AS u(card_id, ordinality)
            JOIN tbl_flashcards f ON f.id = u.card_id
            WHERE f.is_deleted = 'N'
            ORDER BY u.ordinality;
        `
		rows, err := db.Query(ctx, sql, questionIDs)
		if err != nil {
			logger.Error("fetch question details failed", zap.Error(err), zap.Any("questionIDs", questionIDs))
			return nil, errors.New(api.SomeThingWentWrong)
		}
		defer rows.Close()

		var details []FlashCardDetails
		for rows.Next() {
			var (
				idInt     int64
				front     string
				back      string
				choices   []string
				status    string
				createAt  time.Time
				ownerName string
				ord       int64
			)
			if err := rows.Scan(&idInt, &front, &back, &choices, &status, &createAt, &ownerName, &ord); err != nil {
				logger.Error("scan question detail failed", zap.Error(err))
				return nil, errors.New(api.SomeThingWentWrong)
			}
			details = append(details, FlashCardDetails{
				Id:        decimal.NewFromInt(idInt),
				Front:     front,
				Back:      back,
				Choices:   choices,
				Status:    status,
				CreateAt:  createAt,
				OwnerName: ownerName,
				Seq:       decimal.NewFromInt(ord),
			})
		}
		if err := rows.Err(); err != nil {
			logger.Error("iterating details failed", zap.Error(err))
			return nil, errors.New(api.SomeThingWentWrong)
		}
		return details, nil
	}
}

type GetExamSessionFunc func(ctx context.Context, logger *zap.Logger, examID int) (*ExamSessionDto, error)

func NewGetExamSession(db *pgxpool.Pool) GetExamSessionFunc {
	return func(ctx context.Context, logger *zap.Logger, examID int) (*ExamSessionDto, error) {
		const sql = `
		SELECT
			es.question_ids,
			es.is_submitted,
			es.score,
			es.expires_at,
			es.user_id,
			u.username,
			es.answers
		FROM tbl_exam_sessions es
		JOIN tbl_users u ON es.user_id = u.id
		WHERE es.id = $1
		  AND es.is_deleted = 'N'
		`

		row := db.QueryRow(ctx, sql, examID)
		var s ExamSessionDto
		if err := row.Scan(
			&s.QuestionIDs,
			&s.IsSubmitted,
			&s.Score,
			&s.ExpiresAt,
			&s.OwnerID,
			&s.OwnerName,
			&s.Answers,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, errors.New("exam session not found")
			}
			logger.Error("scan exam session failed", zap.Error(err))
			return nil, errors.New(api.SomeThingWentWrong)
		}
		return &s, nil
	}
}

type UpdateExamSessionFunc func(ctx context.Context, logger *zap.Logger, req ExamSessionUpdateRequest) error

func NewUpdateExamSession(db *pgxpool.Pool) UpdateExamSessionFunc {
	return func(ctx context.Context, logger *zap.Logger, req ExamSessionUpdateRequest) error {
		var (
			setClauses []string
			args       []interface{}
			idx        = 1
		)
		if req.Answers != nil {
			setClauses = append(setClauses, fmt.Sprintf("answers      = $%d", idx))
			args = append(args, req.ToJson())
			idx++
		}
		if req.IsSubmitted != nil {
			setClauses = append(setClauses, fmt.Sprintf("is_submitted = $%d", idx))
			args = append(args, *req.IsSubmitted)
			idx++
		}
		if req.Score != nil {
			setClauses = append(setClauses, fmt.Sprintf("score        = $%d", idx))
			args = append(args, *req.Score)
			idx++
		}
		setClauses = append(setClauses, fmt.Sprintf("update_by    = $%d", idx))
		args = append(args, req.Username)
		idx++
		setClauses = append(setClauses, "update_at    = now()")

		sql := fmt.Sprintf(`
            UPDATE tbl_exam_sessions
               SET %s
             WHERE id = $%d
        `, strings.Join(setClauses, ",\n                 "), idx)
		args = append(args, req.Id.IntPart())

		if _, err := db.Exec(ctx, sql, args...); err != nil {
			logger.Error("failed to update exam session", zap.Error(err))
			return errors.New(api.SomeThingWentWrong)
		}
		return nil
	}
}
