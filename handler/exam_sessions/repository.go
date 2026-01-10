package exam_sessions

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"go.uber.org/zap"
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
			rows, err := db.Query(ctx, sql, req.SetId.IntPart(), req.QuestionCount.IntPart())
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
	userId string,
	req StartExamRequest,
	cardIds []int64,
) (int64, error)

func NewInsertStartExamSessions(db *pgxpool.Pool) InsertStartExamSessionsFunc {
	return func(
		ctx context.Context,
		logger *zap.Logger,
		userIdToken string,
		req StartExamRequest,
		cardIds []int64,
	) (int64, error) {
		tx, err := db.Begin(ctx)
		if err != nil {
			logger.Error("failed to begin tx", zap.Error(err))
			return 0, errors.New(api.SomeThingWentWrong)
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
		const sql = `
			insert into tbl_exam_sessions (user_id_token, mode, source_set_id, plan_id, total_questions, time_limit_sec, status,started_at, expires_at)
			values ($1,$2,$3,$4,$5,$6,$7,now(),$8)
			RETURNING id;
		`
		var sessionId int64
		err = tx.QueryRow(ctx, sql,
			userIdToken,
			req.Mode,
			req.SetId,
			req.DailyPlanId,
			req.QuestionCount,
			req.TimeLimitSeconds,
			"ACTIVE",
			req.TimeLimit,
		).Scan(&sessionId)
		if err != nil {
			logger.Error("insert exam session failed", zap.Error(err), zap.String("userIdToken", userIdToken))
			return 0, errors.New(api.SomeThingWentWrong)
		}
		questionType := "MCQ"
		switch req.Mode {
		case "TYPING":
			questionType = "TYPING"
		case "LISTENING":
			questionType = "LISTENING"
		case "SPEAKING":
			questionType = "SPEAKING"
		case "MIXED", "MCQ":
			questionType = "MCQ"
		}

		const insertQuestionsSQL = `
			WITH input AS (
			  SELECT
				x.card_id,
				x.ord::int AS seq
			  FROM unnest($2::bigint[]) WITH ORDINALITY AS x(card_id, ord)
			),
			cards AS (
			  SELECT
				i.seq,
				c.id AS card_id,
				c.front,
				c.back,
				c.choices
			  FROM input i
			  JOIN tbl_flashcards c
				ON c.id = i.card_id
			   AND c.is_deleted = 'N'
			)
			INSERT INTO tbl_exam_questions (
			  session_id,
			  seq,
			  card_id,
			  question_type,
			  front_snapshot,
			  back_snapshot,
			  choices_snapshot,
			  prompt_tts_cache_id,
			  score_max
			)
			SELECT
			  $1,
			  cards.seq,
			  cards.card_id,
			  $3,
			  cards.front,
			  cards.back,
			  cards.choices,
			  NULL,
			  1
			FROM cards
			ORDER BY cards.seq;
			`

		cmdTag, err := tx.Exec(ctx, insertQuestionsSQL, sessionId, cardIds, questionType)
		if err != nil {
			logger.Error("insert exam questions failed",
				zap.Error(err),
				zap.Int64("sessionId", sessionId),
				zap.String("userIdToken", userIdToken),
			)
			return 0, errors.New(api.SomeThingWentWrong)
		}

		if cmdTag.RowsAffected() != int64(len(cardIds)) {
			logger.Error("insert exam questions rows mismatch",
				zap.Int64("sessionId", sessionId),
				zap.Int64("rowsAffected", cmdTag.RowsAffected()),
				zap.Int("expected", len(cardIds)),
			)
			return 0, errors.New(fmt.Sprintf("%s: some cards not found", api.InvalidateBody))
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

type ListExamHistoryFunc func(ctx context.Context, logger *zap.Logger, page, size decimal.Decimal, userIdToken, searchBy string) (ExamSessionListResponse, error)

func NewListExamHistory(db *pgxpool.Pool) ListExamHistoryFunc {
	return func(ctx context.Context, logger *zap.Logger, page, size decimal.Decimal, userIdToken, searchBy string) (ExamSessionListResponse, error) {
		resp := ExamSessionListResponse{}
		pageInt := int(page.IntPart())
		sizeInt := int(size.IntPart())
		offset := (pageInt - 1) * sizeInt

		pattern := "%" + searchBy + "%"
		sqlCount := `
			select count(*) from tbl_exam_sessions
			where user_id_token=$1
		`
		err := db.QueryRow(ctx, sqlCount, userIdToken).Scan(&resp.TotalElements)
		if err != nil {
			logger.Error("failed to list exam sessions", zap.Error(err))
			return ExamSessionListResponse{}, err
		}
		totalPageFloat, _ := resp.TotalElements.Div(size).Float64()
		resp.TotalPage = decimal.NewFromFloat(math.Ceil(totalPageFloat))
		sqlRows := `
			select id,mode,source_set_id,
			       plan_id, total_questions, time_limit_sec,
			       CASE 
					WHEN expires_at IS NOT NULL AND expires_at < NOW() THEN 'EXPIRED'
					ELSE status END  as current_status,
			    started_at, expires_at, submitted_at, score_total, score_max, create_at from tbl_exam_sessions
			where user_id_token=$1 
			AND (mode || ' ' || status) LIKE $2
			ORDER BY create_at DESC
			 OFFSET $3 LIMIT $4
		`
		rows, err := db.Query(ctx, sqlRows, userIdToken, pattern, offset, sizeInt)
		if err != nil {
			return ExamSessionListResponse{}, err
		}
		examList := []ExamSessionListResponseDetails{}
		for rows.Next() {
			var item ExamSessionListResponseDetails
			err := rows.Scan(
				&item.ID,
				&item.Mode,
				&item.SourceSetID,
				&item.PlanID,
				&item.TotalQuestions,
				&item.TimeLimitSec,
				&item.Status,
				&item.StartedAt,
				&item.ExpiresAt,
				&item.SubmittedAt,
				&item.ScoreTotal,
				&item.ScoreMax,
				&item.CreatedAt,
			)
			if err != nil {
				return resp, err
			}
			examList = append(examList, item)
		}

		resp.Content = examList

		return resp, nil
	}

}
