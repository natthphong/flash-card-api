package flashcard_sets

import (
	"bytes"
	"encoding/base64"
	"encoding/csv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

func NewFlashCardSetsImportCsvHandler(
	insertFlashCardsSetFunc InsertFlashCardsSetFunc,
) fiber.Handler {

	return func(c *fiber.Ctx) error {
		var req FlashCardsSetsCsvImportRequest
		ctx := c.Context()
		logger := logz.NewLogger()
		requestId := c.Get("requestId")

		// 1) parse + validate JSON
		if err := c.BodyParser(&req); err != nil {
			logger.Error("body parse error", zap.String("requestId", requestId), zap.Error(err))
			return api.BadRequest(c, api.InvalidateBody)
		}
		if err := req.Validate(); err != nil {
			logger.Error("validation error", zap.String("requestId", requestId), zap.Error(err))
			return api.BadRequest(c, err.Error())
		}

		// 2) identity from middleware
		req.UserId = c.Locals("userId").(string)
		userIdToken := c.Locals("userIdToken").(string)
		req.OwnerIdToken = userIdToken

		// 3) decode CSV
		csvBytes, decodeErr := base64.StdEncoding.DecodeString(req.File)
		if decodeErr != nil {
			logger.Error("decode base64", zap.String("requestId", requestId), zap.Error(decodeErr))
			return api.BadRequest(c, "file must be a valid base64‑encoded CSV")
		}
		reader := csv.NewReader(bytes.NewReader(csvBytes))
		reader.TrimLeadingSpace = true
		if req.CommandRec != "" {
			reader.Comma = []rune(req.CommandRec)[0] // custom delimiter
		}

		records, csvErr := reader.ReadAll()
		if csvErr != nil {
			logger.Error("read csv", zap.String("requestId", requestId), zap.Error(csvErr))
			return api.BadRequest(c, "invalid CSV content")
		}
		if len(records) == 0 {
			return api.BadRequest(c, "CSV must have at least one row")
		}

		// 4) convert rows -> cards
		var cards []InsertFlashCards
		for idx, rec := range records {
			if len(rec) < 2 { // need at least front & back
				logger.Warn("csv row skipped (need front & back)",
					zap.String("requestId", requestId), zap.Int("row", idx+1))
				continue
			}

			front := strings.TrimSpace(rec[0])
			back := strings.TrimSpace(rec[1])

			var choices []string
			if len(rec) >= 3 && strings.TrimSpace(rec[2]) != "" {
				raw := strings.Trim(strings.TrimSpace(rec[2]), "{}")
				for _, ch := range strings.Split(raw, ",") {
					val := strings.TrimSpace(strings.Trim(ch, `"'`))
					if val != "" {
						choices = append(choices, val)
					}
				}
			}

			cards = append(cards, InsertFlashCards{
				Front:   front,
				Back:    back,
				Choices: choices,
			})
		}
		if len(cards) == 0 {
			return api.BadRequest(c, "CSV contained no valid rows")
		}

		// 5) reuse existing InsertFlashCardsSetFunc
		createReq := FlashCardSetsCreateRequest{
			Title:        req.Title,
			Description:  req.Description,
			IsPublic:     req.IsPublic,
			FlashCards:   &cards,
			OwnerIdToken: req.OwnerIdToken,
			UserId:       req.UserId,
		}
		if err := insertFlashCardsSetFunc(ctx, logger, createReq); err != nil {
			return api.InternalError(c, api.SomeThingWentWrong)
		}
		return api.Ok(c, nil)
	}
}
