package voice

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/adapter"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

func NewPronunciationScoreHandler(
	homeProxyAdapter adapter.Adapter,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()
		var resp SttResponse
		logger := logz.NewLogger()
		_ = logger
		requestId := c.Get("requestId")
		fh, err := c.FormFile("file")
		if err != nil || fh == nil {
			return api.BadRequest(c, "missing file")
		}

		mediaType := c.FormValue("mediaType")
		sourceText := c.FormValue("sourceText")
		if sourceText == "" {
			return api.BadRequest(c, "missing sourceText")
		}
		if mediaType == "" {
			mediaType = mime.TypeByExtension(strings.ToLower(filepath.Ext(fh.Filename)))
			if mediaType == "" {
				mediaType = "application/octet-stream"
			}
		}
		if !isAllowedMedia(mediaType) {
			return api.BadRequest(c, "unsupported mediaType")
		}

		src, err := fh.Open()
		if err != nil {
			return api.InternalError(c, err.Error())
		}
		defer src.Close()
		audioBytes, err := io.ReadAll(src)
		if err != nil {
			return api.InternalError(c, err.Error())
		}

		_, body, err := homeProxyAdapter.Post(ctx, api.SttPath, &adapter.RequestOptions{
			Headers: map[string]string{"requestId": requestId},
			Form: &adapter.FormData{
				Fields: map[string]string{
					"mediaType": mediaType,
				},
				Files: []adapter.FormFile{
					{
						FieldName:   "file",
						FileName:    fh.Filename,
						Reader:      bytes.NewReader(audioBytes),
						ContentType: mediaType,
					},
				},
			},
		})
		err = json.Unmarshal(body, &resp)
		if err != nil {
			logger.Error(err.Error())
			logger.Error("Failed to process audio", zap.Any("body", string(body)))
			return err
		}
		logger.Info("afterCallAPI", zap.Any("resp", resp))
		// TODO insert tbl_pronunciation_attempt

		sttText := resp.Body.Text
		report := ScoreByWER(sourceText, sttText)
		return api.Ok(c, fiber.Map{
			"sourceText": sourceText,
			"sttText":    sttText,
			"score":      report.Score,
			"wer":        report.WER,
			"details":    report,
		})
	}
}

func isAllowedMedia(mt string) bool {
	return strings.HasPrefix(mt, "audio/") ||
		mt == "application/octet-stream"
}
