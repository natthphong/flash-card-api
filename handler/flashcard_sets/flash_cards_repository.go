package flashcard_sets

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"go.uber.org/zap"
)

type InsertFlashCardsFunc func(ctx context.Context, logger *zap.Logger, flashCards FlashCardsCreateRequest) error

func NewInsertFlashCards(db *pgxpool.Pool) InsertFlashCardsFunc {
	return func(ctx context.Context, logger *zap.Logger, flashCards FlashCardsCreateRequest) (err error) {
		// begin transaction
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

		cards := flashCards.Cards
		n := len(cards)
		if n == 0 {
			return nil
		}
		var (
			valueStrings []string
			valueArgs    []interface{}
		)
		for i, card := range cards {
			finalChoices := card.Choices
			if len(finalChoices) < 4 {
				finalChoices = GetChoices(toPointerSlice(cards), &card)
			}

			off := i * 7
			valueStrings = append(valueStrings, fmt.Sprintf(
				"($%d,$%d,$%d,$%d,$%d,$%d,$%d)",
				off+1, off+2, off+3, off+4, off+5, off+6, off+7,
			))
			valueArgs = append(valueArgs,
				flashCards.SetId,
				card.Front,
				card.Back,
				finalChoices,
				CardStatusStudying,
				flashCards.UserId,
				i,
			)
		}

		sql := fmt.Sprintf(`
            INSERT INTO tbl_flashcards
                (set_id, front, back, choices, status, create_by, seq)
            VALUES %s
        `, strings.Join(valueStrings, ","))

		if _, err = tx.Exec(ctx, sql, valueArgs...); err != nil {
			logger.Error("batch insert flashcards failed",
				zap.Error(err), zap.Any("set_id", flashCards.SetId))
			return errors.New(api.SomeThingWentWrong)
		}

		return nil
	}
}

// --- Update func ---

type UpdateFlashCardsFunc func(ctx context.Context, logger *zap.Logger, req FlashCardsUpdateRequest) error

func NewUpdateFlashCards(db *pgxpool.Pool) UpdateFlashCardsFunc {
	return func(ctx context.Context, logger *zap.Logger, req FlashCardsUpdateRequest) error {
		if err := req.Validate(); err != nil {
			return err
		}

		var (
			setClauses []string
			args       []interface{}
			idx        = 1
		)

		if req.Front != nil {
			setClauses = append(setClauses, fmt.Sprintf("front     = $%d", idx))
			args = append(args, *req.Front)
			idx++
		}
		if req.Back != nil {
			setClauses = append(setClauses, fmt.Sprintf("back      = $%d", idx))
			args = append(args, *req.Back)
			idx++
		}
		if req.Choices != nil {
			setClauses = append(setClauses, fmt.Sprintf("choices   = $%d", idx))
			args = append(args, *req.Choices)
			idx++
		}
		if req.Status != nil {
			setClauses = append(setClauses, fmt.Sprintf("status    = $%d", idx))
			args = append(args, *req.Status)
			idx++
		}
		// audit fields
		setClauses = append(setClauses, fmt.Sprintf("update_by = $%d", idx))
		args = append(args, req.UserId)
		idx++
		setClauses = append(setClauses, "update_at  = now()")

		sql := fmt.Sprintf(`
            UPDATE tbl_flashcards
               SET %s
             WHERE id = $%d
        `, strings.Join(setClauses, ",\n                 "), idx)
		args = append(args, req.Id)

		if _, err := db.Exec(ctx, sql, args...); err != nil {
			logger.Error("failed to update flashcard", zap.Error(err))
			return errors.New(api.SomeThingWentWrong)
		}
		return nil
	}
}

type DeleteFlashCardsFunc func(ctx context.Context, logger *zap.Logger, req FlashCardsDeleteRequest) error

func NewDeleteFlashCards(db *pgxpool.Pool) DeleteFlashCardsFunc {
	return func(ctx context.Context, logger *zap.Logger, req FlashCardsDeleteRequest) error {
		if err := req.Validate(); err != nil {
			return err
		}

		const sql = `
            UPDATE tbl_flashcards
               SET is_deleted = 'Y',
                   update_by  = $1,
                   update_at  = now()
             WHERE id = $2
        `
		if _, err := db.Exec(ctx, sql, req.UserId, req.Id); err != nil {
			logger.Error("failed to delete flashcard", zap.Error(err))
			return errors.New(api.SomeThingWentWrong)
		}
		return nil
	}
}
