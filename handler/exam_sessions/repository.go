package exam_sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
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
				WHERE set_id = $1
				  AND is_deleted = 'N'
				ORDER BY random()
				LIMIT $2;
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
			insert into tbl_exam_sessions (user_id_token, mode, source_set_id, plan_id, total_questions, time_limit_sec, status,started_at, expires_at,score_max)
			values ($1,$2,$3,$4,$5,$6,$7,now(),$8,$9)
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
			req.QuestionCount,
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

type GetExamSessionFunc func(ctx context.Context, logger *zap.Logger, examID int64) (*ExamSessionDto, error)

func NewGetExamSession(db *pgxpool.Pool) GetExamSessionFunc {
	return func(ctx context.Context, logger *zap.Logger, examID int64) (*ExamSessionDto, error) {
		// 1) Load session header
		const sqlSession = `
			SELECT
			  id,
			  CASE 
					WHEN expires_at IS NOT NULL AND expires_at < NOW() THEN 'EXPIRED'
			ELSE status END  as current_status,
			  mode,
			  total_questions,
			  score_total,
			  score_max,
			  expires_at,
			  submitted_at,
			  create_at
			FROM tbl_exam_sessions
			WHERE id = $1;
			`

		var dto ExamSessionDto
		err := db.QueryRow(ctx, sqlSession, examID).Scan(
			&dto.ID,
			&dto.Status,
			&dto.Mode,
			&dto.TotalQuestions,
			&dto.ScoreTotal,
			&dto.ScoreMax,
			&dto.ExpiresAt,
			&dto.SubmittedAt,
			&dto.CreatedAt,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, errors.New("exam session not found")
			}
			logger.Error("scan exam session header failed", zap.Error(err), zap.Int64("examID", examID))
			return nil, errors.New(api.SomeThingWentWrong)
		}

		// 2) Load questions + answers
		const sqlQA = `
			SELECT
			   txq.id as question_id,
			   txq.seq,
			   txq.card_id,
			   txq.question_type,
			   txq.front_snapshot,
			   txq.back_snapshot,
			   txq.choices_snapshot,
			   txq.prompt_tts_cache_id,
			   txq.score_max,
			
			   txa.id as answer_id,
			   txa.selected_choice,
			   txa.typed_text,
			   txa.audio_url,
			   txa.recognized_text,
			   txa.pronunciation_score,
			   txa.is_correct,
			   txa.score_awarded,
			   txa.answered_at,
			   txa.detail
			FROM tbl_exam_questions txq
			LEFT JOIN tbl_exam_answers txa
			  ON txq.id = txa.question_id
			 AND txq.session_id = txa.session_id
			WHERE txq.session_id = $1
			ORDER BY txq.seq ASC;
		`

		rows, err := db.Query(ctx, sqlQA, examID)
		if err != nil {
			logger.Error("query exam questions failed", zap.Error(err), zap.Int64("examID", examID))
			return nil, errors.New(api.SomeThingWentWrong)
		}
		defer rows.Close()
		dto.Questions = make([]ExamQuestionDto, 0, dto.TotalQuestions)
		for rows.Next() {
			var q ExamQuestionDto

			var promptTtsCacheId *int64
			var choices []string

			// Answer nullable fields
			var ansId *int64
			var selectedChoice *string
			var typedText *string
			var audioURL *string
			var recognizedText *string
			var pronunciationScore *int
			var isCorrect *string
			var scoreAwarded *int
			var answeredAt *time.Time
			var detail *json.RawMessage

			if err := rows.Scan(
				&q.QuestionID,
				&q.Seq,
				&q.CardID,
				&q.QuestionType,
				&q.FrontSnapshot,
				&q.BackSnapshot,
				&choices,
				&promptTtsCacheId,
				&q.ScoreMax,

				&ansId,
				&selectedChoice,
				&typedText,
				&audioURL,
				&recognizedText,
				&pronunciationScore,
				&isCorrect,
				&scoreAwarded,
				&answeredAt,
				&detail,
			); err != nil {
				logger.Error("scan exam question row failed", zap.Error(err), zap.Int64("examID", examID))
				return nil, errors.New(api.SomeThingWentWrong)
			}

			q.ChoicesSnapshot = choices
			q.PromptTtsCacheId = promptTtsCacheId

			if ansId != nil {
				a := &ExamAnswerDto{
					AnswerID:           *ansId,
					SelectedChoice:     selectedChoice,
					TypedText:          typedText,
					AudioURL:           audioURL,
					RecognizedText:     recognizedText,
					PronunciationScore: pronunciationScore,
					IsCorrect:          isCorrect,
					ScoreAwarded:       scoreAwarded,
					AnsweredAt:         answeredAt,
				}
				if detail != nil {
					a.Detail = *detail
				}
				q.Answer = a
			}

			dto.Questions = append(dto.Questions, q)
		}

		if rows.Err() != nil {
			logger.Error("iterate exam question rows failed", zap.Error(rows.Err()), zap.Int64("examID", examID))
			return nil, errors.New(api.SomeThingWentWrong)
		}

		return &dto, nil
	}
}

type UpdateExamSessionFunc func(ctx context.Context, logger *zap.Logger, req ExamSessionUpdateRequest) (string, error)

func NewUpdateExamSession(db *pgxpool.Pool) UpdateExamSessionFunc {
	return func(ctx context.Context, logger *zap.Logger, req ExamSessionUpdateRequest) (string, error) {
		// begin transaction
		var isCorrect bool
		var questionId int64
		score := 0
		correct := utils.FlagN
		tx, err := db.Begin(ctx)
		if err != nil {
			logger.Error("failed to begin tx", zap.Error(err))
			return correct, errors.New(api.SomeThingWentWrong)
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
		choiceIndex := utils.GetIndexFromString(req.Choice)
		sqlQuestion :=
			`
			SELECT
				  id,
				  (
					$1 BETWEEN 1 AND array_length(choices_snapshot, 1)
					AND choices_snapshot[$1] = back_snapshot
				  ) AS correct
				FROM tbl_exam_questions
				WHERE session_id = $2
				  AND seq = $3;
			`
		err = tx.QueryRow(ctx, sqlQuestion, choiceIndex, req.SessionId, req.SeqId).Scan(
			&questionId, &isCorrect)
		if err != nil {
			logger.Error("failed to update exam question row", zap.Error(err))
			return correct, errors.New(api.SomeThingWentWrong)
		}

		if isCorrect {
			score = 1
			correct = utils.FlagY
		}

		sqlInsertAnswer :=
			`
			INSERT INTO tbl_exam_answers (
				  session_id,
				  question_id,
				  user_id_token,
				  selected_choice,
				  typed_text,
				  is_correct,
				  score_awarded,
				  answered_at
				)
				VALUES (
				  $1, $2, $3, $4, $5, $6, $7, now()
				)
				ON CONFLICT (session_id, question_id)
				DO UPDATE SET
				  selected_choice = EXCLUDED.selected_choice,
				  typed_text      = EXCLUDED.typed_text,
				  is_correct      = EXCLUDED.is_correct,
				  score_awarded   = EXCLUDED.score_awarded,
				  answered_at     = now();

        `
		_, err = tx.Exec(ctx, sqlInsertAnswer, req.SessionId, questionId, req.UserIdToken, req.Choice, req.AnswerType, correct, score)
		if err != nil {
			logger.Error("failed to update exam answer row", zap.Error(err))
			return correct, errors.New(api.SomeThingWentWrong)
		}
		return correct, nil
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

type InsertReviewLogsFunc func(
	ctx context.Context,
	logger *zap.Logger,
	tx pgx.Tx,
	userIdToken string,
	items []ExamSubmitItem,
) error

func NewInsertReviewLogsFunc() InsertReviewLogsFunc {
	return func(ctx context.Context, logger *zap.Logger, tx pgx.Tx, userIdToken string, items []ExamSubmitItem) error {
		if len(items) == 0 {
			return nil
		}

		cardIDs := make([]int64, 0, len(items))
		sources := make([]string, 0, len(items))
		grades := make([]int16, 0, len(items))
		isCorrects := make([]string, 0, len(items))
		answerDetails := make([][]byte, 0, len(items))
		for _, it := range items {
			cardIDs = append(cardIDs, it.CardID)
			sources = append(sources, it.Source)
			grades = append(grades, it.Grade)
			isCorrects = append(isCorrects, it.IsCorrect)
			answerDetails = append(answerDetails, []byte(it.AnswerDetail))
		}

		const sql = `
			INSERT INTO tbl_review_log (
				user_id_token,
				card_id,
				source,
				grade,
				is_correct,
				answer_detail
			)
			SELECT
				$1::varchar(36),
				x.card_id,
				x.source,
				x.grade,
				x.is_correct,
				x.answer_detail::jsonb
			FROM unnest(
				$2::bigint[],
				$3::text[],
				$4::smallint[],
				$5::text[],
				$6::jsonb[]
			) AS x(card_id, source, grade, is_correct, answer_detail);
		`

		_, err := tx.Exec(ctx, sql, userIdToken, cardIDs, sources, grades, isCorrects, answerDetails)
		return err
	}
}

type UpdateExamSessionAfterSubmitFunc func(
	ctx context.Context,
	logger *zap.Logger,
	tx pgx.Tx,
	sessionId int64,
	score int,
) error

func NewUpdateExamSessionAfterSubmit() UpdateExamSessionAfterSubmitFunc {
	return func(ctx context.Context, logger *zap.Logger, tx pgx.Tx, sessionId int64, score int) error {
		sql := `
			update tbl_exam_sessions set submitted_at = now(), score_total = $2 where id = $1 
		`
		_, err := tx.Exec(ctx, sql, sessionId, score)
		return err
	}
}

type UpsertUserFlashcardSrsBatchFunc func(
	ctx context.Context,
	logger *zap.Logger,
	tx pgx.Tx,
	userIdToken string,
	items []ExamSubmitItem,
) error

func NewUpsertUserFlashcardSrsBatchFunc() UpsertUserFlashcardSrsBatchFunc {
	return func(ctx context.Context, logger *zap.Logger, tx pgx.Tx, userIdToken string, items []ExamSubmitItem) error {
		if len(items) == 0 {
			return nil
		}

		cardIDs := make([]int64, 0, len(items))
		grades := make([]int16, 0, len(items))

		for _, it := range items {
			cardIDs = append(cardIDs, it.CardID)
			grades = append(grades, it.Grade)
		}

		const sql = `
		WITH input AS (
		  SELECT
			$1::varchar(36) AS user_id_token,
			x.card_id::bigint AS card_id,
			x.grade::smallint AS grade
		  FROM unnest($2::bigint[], $3::smallint[]) AS x(card_id, grade)
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

		_, err := tx.Exec(ctx, sql, userIdToken, cardIDs, grades)
		return err
	}
}
