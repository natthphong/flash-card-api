package flashcard_sets

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

type InsertFlashCardsSetFunc func(ctx context.Context, logger *zap.Logger, sets FlashCardSetsCreateRequest) error

func NewInsertFlashCardsSet(db *pgxpool.Pool) InsertFlashCardsSetFunc {
	return func(ctx context.Context, logger *zap.Logger, sets FlashCardSetsCreateRequest) (err error) {
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

		const insertSetSQL = `
            INSERT INTO tbl_flashcard_sets
                (owner_id, title, description, is_public, create_by)
            VALUES ($1,$2,$3,$4,$5)
            RETURNING id
        `
		var setID int
		if err = tx.QueryRow(ctx, insertSetSQL,
			sets.OwnerId, sets.Title, sets.Description, sets.IsPublic, sets.Username,
		).Scan(&setID); err != nil {
			logger.Error("failed to insert flashcard_sets", zap.Error(err))
			return errors.New(api.SomeThingWentWrong)
		}

		// 2) batch‐insert all cards, if any
		if sets.FlashCards != nil && len(*sets.FlashCards) > 0 {
			cards := *sets.FlashCards

			// build placeholders and args
			var (
				valueStrings []string
				valueArgs    []interface{}
			)
			// each card has 7 columns: set_id, front, back, choices, status, create_by, seq
			for i, card := range cards {
				finalChoices := card.Choices
				if len(finalChoices) < 4 {
					finalChoices = GetChoices(toPointerSlice(cards), &card)
				}

				// For row i we need placeholders ($1,$2...$7), row i+1 ($8,$9...$14), etc.
				offset := i * 7
				placeholders := fmt.Sprintf(
					"($%d,$%d,$%d,$%d,$%d,$%d,$%d)",
					offset+1, offset+2, offset+3, offset+4, offset+5, offset+6, offset+7,
				)
				valueStrings = append(valueStrings, placeholders)

				valueArgs = append(valueArgs,
					setID,
					card.Front,
					card.Back,
					finalChoices,
					CardStatusStudying,
					sets.Username,
					i, // seq
				)
			}

			// assemble and execute single INSERT
			insertCardsSQL := fmt.Sprintf(`
                INSERT INTO tbl_flashcards
                    (set_id, front, back, choices, status, create_by, seq)
                VALUES %s
            `, strings.Join(valueStrings, ","))
			if _, err = tx.Exec(ctx, insertCardsSQL, valueArgs...); err != nil {
				logger.Error("batch insert flashcards failed",
					zap.Error(err), zap.Int("set_id", setID))
				return errors.New(api.SomeThingWentWrong)
			}
		}

		return nil
	}
}

type DuplicateFlashCardsSetFunc func(
	ctx context.Context,
	logger *zap.Logger,
	req DuplicateFlashCardsSetRequest,
) (newSetID int, err error)

func NewDuplicateFlashCardsSet(db *pgxpool.Pool) DuplicateFlashCardsSetFunc {
	return func(
		ctx context.Context,
		logger *zap.Logger,
		req DuplicateFlashCardsSetRequest,
	) (newSetID int, err error) {

		tx, err := db.Begin(ctx)
		if err != nil {
			logger.Error("tx begin failed", zap.Error(err))
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
		const dupSetSQL = `
		INSERT INTO tbl_flashcard_sets
				(owner_id, title, description, is_public, create_by)
		SELECT  $1       , title, description, is_public, $2
		FROM    tbl_flashcard_sets
		WHERE   id = $3
		RETURNING id;
`
		if err = tx.QueryRow(
			ctx, dupSetSQL,
			req.OwnerID,
			req.Username,
			req.OldSetID,
		).Scan(&newSetID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return 0, errors.New(api.NotFound)
			}
			logger.Error("duplicate set failed", zap.Error(err))
			return 0, errors.New(api.SomeThingWentWrong)
		}

		const dupCardsSQL = `
			INSERT INTO tbl_flashcards
					(set_id, front, back, choices, status, create_by, seq)
			SELECT  $1      , front, back, choices, status, $2       , seq
			FROM    tbl_flashcards
			WHERE   set_id = $3;
		`
		if _, err = tx.Exec(
			ctx, dupCardsSQL,
			newSetID,
			req.Username,
			req.OldSetID,
		); err != nil {
			logger.Error("duplicate cards failed", zap.Error(err), zap.Int("new_set_id", newSetID))
			return 0, errors.New(api.SomeThingWentWrong)
		}
		return newSetID, nil
	}
}

type InsertAndMergeFlashCardSetsTrackerFunc func(
	ctx context.Context,
	logger *zap.Logger,
	row FlashCardSetsTrackerUpsert,
) error

func NewInsertAndMergeFlashCardSetsTracker(
	db *pgxpool.Pool,
) InsertAndMergeFlashCardSetsTrackerFunc {
	const upsertSQL = `
		INSERT INTO public.tbl_flashcard_sets_tracker
				(set_id, user_id, card_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, set_id)
		DO UPDATE
		   SET card_id = EXCLUDED.card_id;    
		`

	return func(
		ctx context.Context,
		logger *zap.Logger,
		row FlashCardSetsTrackerUpsert,
	) (err error) {

		tx, err := db.Begin(ctx)
		if err != nil {
			logger.Error("tx begin failed", zap.Error(err))
			return errors.New(api.SomeThingWentWrong)
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback(ctx)
			} else if cmErr := tx.Commit(ctx); cmErr != nil {
				logger.Error("tx commit failed", zap.Error(cmErr))
				err = errors.New(api.SomeThingWentWrong)
			}
		}()

		if _, err = tx.Exec(
			ctx, upsertSQL,
			row.SetID,
			row.OwnerID,
			row.CardID,
		); err != nil {
			logger.Error("upsert tracker failed", zap.Error(err))
			return errors.New(api.SomeThingWentWrong)
		}

		return nil
	}
}

type ResetStatusFlashCardsFunc func(
	ctx context.Context,
	logger *zap.Logger,
	req ResetFlashCardStatusRequest,
) error

func NewResetStatusFlashCards(db *pgxpool.Pool) ResetStatusFlashCardsFunc {
	const updateSQL = `
		UPDATE tbl_flashcards
		SET    status = $1
		WHERE  set_id = $2;
	`

	return func(
		ctx context.Context,
		logger *zap.Logger,
		req ResetFlashCardStatusRequest,
	) (err error) {

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
		if _, err = tx.Exec(ctx, updateSQL, req.Status, req.SetID); err != nil {
			logger.Error("reset status failed",
				zap.Error(err), zap.Any("set_id", req.SetID))
			return errors.New(api.SomeThingWentWrong)
		}

		return nil
	}
}

func toPointerSlice(in []InsertFlashCards) []*InsertFlashCards {
	out := make([]*InsertFlashCards, len(in))
	for i := range in {
		out[i] = &in[i]
	}
	return out
}

type UpdateFlashCardSetsFunc func(ctx context.Context, logger *zap.Logger, req FlashCardSetsUpdateRequest) error

func NewUpdateFlashCardSets(db *pgxpool.Pool) UpdateFlashCardSetsFunc {
	return func(ctx context.Context, logger *zap.Logger, req FlashCardSetsUpdateRequest) error {
		if err := req.Validate(); err != nil {
			return err
		}

		var (
			setClauses []string
			args       []interface{}
			idx        = 1
		)
		if req.Title != "" {
			setClauses = append(setClauses, fmt.Sprintf("title        = $%d", idx))
			args = append(args, req.Title)
			idx++
		}
		if req.Description != "" {
			setClauses = append(setClauses, fmt.Sprintf("description  = $%d", idx))
			args = append(args, req.Description)
			idx++
		}
		if req.IsPublic != "" {
			setClauses = append(setClauses, fmt.Sprintf("is_public    = $%d", idx))
			args = append(args, req.IsPublic)
			idx++
		}
		setClauses = append(setClauses, fmt.Sprintf("update_by    = $%d", idx))
		args = append(args, req.Username)
		idx++
		setClauses = append(setClauses, "update_at    = now()")

		sql := fmt.Sprintf(`
            UPDATE tbl_flashcard_sets
               SET %s
             WHERE id = $%d
        `, strings.Join(setClauses, ",\n                 "), idx)
		args = append(args, req.Id)

		if _, err := db.Exec(ctx, sql, args...); err != nil {
			logger.Error("failed to update flashcard_sets", zap.Error(err))
			return errors.New(api.SomeThingWentWrong)
		}
		return nil
	}
}

type DeleteFlashCardSetsFunc func(ctx context.Context, logger *zap.Logger, req FlashCardSetsDeleteRequest) error

func NewDeleteFlashCardSets(db *pgxpool.Pool) DeleteFlashCardSetsFunc {
	return func(ctx context.Context, logger *zap.Logger, req FlashCardSetsDeleteRequest) error {
		if err := req.Validate(); err != nil {
			return err
		}

		sql := `
            UPDATE tbl_flashcard_sets
               SET is_deleted = 'Y',
                   update_by  = $1,
                   update_at  = now()
             WHERE id = $2
        `
		if _, err := db.Exec(ctx, sql, req.Username, req.Id); err != nil {
			logger.Error("failed to delete flashcard_sets", zap.Error(err))
			return errors.New(api.SomeThingWentWrong)
		}
		return nil
	}
}

type ListFlashCardSetsFunc func(
	ctx context.Context,
	logger *zap.Logger,
	req FlashCardSetsListRequest,
	ownerId int,
) (FlashCardSetsListResponse, error)

func NewListFlashCardSets(db *pgxpool.Pool) ListFlashCardSetsFunc {
	return func(
		ctx context.Context,
		logger *zap.Logger,
		req FlashCardSetsListRequest,
		ownerId int,
	) (FlashCardSetsListResponse, error) {
		var resp FlashCardSetsListResponse
		page := int(req.Page.IntPart())
		size := int(req.Size.IntPart())
		offset := (page - 1) * size

		pattern := "%" + req.SearchBy + "%"

		const countSQL = `
			SELECT count(*) 
			  FROM tbl_flashcard_sets
			 WHERE (owner_id = $1 OR ($2 = 'N' AND is_public = 'Y'))
			   AND is_public = $3
			   AND (title || ' ' || description) LIKE $4
				and is_deleted = 'N'
		`
		var totalElements int64
		if err := db.QueryRow(ctx, countSQL,
			ownerId, req.IsMine, req.IsPublic, pattern,
		).Scan(&totalElements); err != nil {
			logger.Error("failed to count flashcard_sets", zap.Error(err))
			return resp, errors.New(api.SomeThingWentWrong)
		}

		totalPages := int64(math.Ceil(float64(totalElements) / float64(size)))

		const listSQL = `
			SELECT id , title, description, is_public, owner_id, create_by,
			     (
					SELECT COUNT(*) 
					FROM tbl_flashcards f 
					WHERE f.set_id = s.id
				) AS term    
			  FROM tbl_flashcard_sets s
			 WHERE (owner_id = $1 OR ($2 = 'N' AND is_public = 'Y'))
			   AND is_public = $3
			   AND (title || ' ' || description) LIKE $4
			 	and is_deleted = 'N'
			 ORDER BY id
			 OFFSET $5 LIMIT $6
		`
		rows, err := db.Query(ctx, listSQL,
			ownerId, req.IsMine, req.IsPublic, pattern, offset, size,
		)
		if err != nil {
			logger.Error("failed to list flashcard_sets", zap.Error(err))
			return resp, errors.New(api.SomeThingWentWrong)
		}
		defer rows.Close()

		items := []FlashCardSetsListResponseDetails{}
		for rows.Next() {
			var d FlashCardSetsListResponseDetails
			if err := rows.Scan(
				&d.SetId,
				&d.Title,
				&d.Description,
				&d.IsPublic,
				&d.OwnerId,
				&d.OwnerName,
				&d.Term,
			); err != nil {
				logger.Error("scan flashcard_sets row failed", zap.Error(err))
				return resp, errors.New(api.SomeThingWentWrong)
			}
			items = append(items, d)
		}
		if err := rows.Err(); err != nil {
			logger.Error("rows iteration error", zap.Error(err))
			return resp, errors.New(api.SomeThingWentWrong)
		}

		resp = FlashCardSetsListResponse{
			Content:       items,
			TotalPage:     decimal.NewFromInt(totalPages),
			TotalElements: decimal.NewFromInt(totalElements),
		}
		return resp, nil
	}
}

type FlashCardSetsInquiryFunc func(
	ctx context.Context,
	logger *zap.Logger,
	setID int,
	userID int,
) ([]FlashCardSetsInquiryResponse, error)

func NewFlashCardSetsInquiry(db *pgxpool.Pool) FlashCardSetsInquiryFunc {
	return func(
		ctx context.Context,
		logger *zap.Logger,
		setID int,
		userID int,
	) ([]FlashCardSetsInquiryResponse, error) {
		const inquirySQL = `
            SELECT id, front, back, choices, status, create_at, create_by,seq
              FROM tbl_flashcards
             WHERE is_deleted = 'N'
               AND set_id = $1
            ORDER BY seq,id
        `
		rows, err := db.Query(ctx, inquirySQL, setID)
		if err != nil {
			logger.Error("failed to query flashcards", zap.Error(err), zap.Int("set_id", setID))
			return nil, errors.New(api.SomeThingWentWrong)
		}
		defer rows.Close()

		result := []FlashCardSetsInquiryResponse{}
		for rows.Next() {
			var (
				idInt     int64
				front     string
				back      string
				choices   []string
				status    string
				createAt  time.Time
				ownerName string
				seq       decimal.Decimal
			)
			if err := rows.Scan(
				&idInt,
				&front,
				&back,
				&choices,
				&status,
				&createAt,
				&ownerName,
				&seq,
			); err != nil {
				logger.Error("scan flashcard row failed", zap.Error(err), zap.Int("set_id", setID))
				return nil, errors.New(api.SomeThingWentWrong)
			}

			result = append(result, FlashCardSetsInquiryResponse{
				Id:        decimal.NewFromInt(idInt),
				Front:     front,
				Back:      back,
				Choices:   choices,
				Status:    status,
				CreateAt:  createAt,
				OwnerName: ownerName,
				Seq:       seq,
			})
		}

		if err := rows.Err(); err != nil {
			logger.Error("iterating flashcard rows failed", zap.Error(err), zap.Int("set_id", setID))
			return nil, errors.New(api.SomeThingWentWrong)
		}

		if len(result) != 0 {
			sqlTrack := `
				select card_id from tbl_flashcard_sets_tracker
				where user_id=$1 and set_id = $2
				`
			var cardId decimal.Decimal
			err := db.QueryRow(ctx, sqlTrack, userID, setID).Scan(&cardId)
			if err != nil {
				logger.Error(err.Error())
			}
			if err == nil {
				for i := 0; i < len(result); i++ {
					if result[i].Id.Equal(cardId) {
						result[i].IsCurrent = true
					}
				}
			}
		}

		return result, nil
	}
}
