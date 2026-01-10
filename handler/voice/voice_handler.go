package voice

import (
	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/adapter"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

func NewVoceHandler(
	homeProxyAdapter adapter.Adapter,
	updateHitCacheAndReturnAudio UpdateHitCacheAndReturnAudio,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req VoiceRequest
		ctx := c.Context()
		logger := logz.NewLogger()
		requestId := c.Get("requestId")
		err := c.BodyParser(&req)
		if err != nil {
			return api.InternalError(c, "invalid request")
		}
		validate := validator.New()
		err = validate.Struct(req)
		if err := validate.Struct(req); err != nil {
			return api.ValidationErrorResponse(c, err, req)
		}
		audioUrl, err := updateHitCacheAndReturnAudio(ctx, logger, "")
		if err != nil || audioUrl == "" {
			logger.Info("audio url not found in cache")
			var baseResponse api.Response
			var bodyTtsResponse TtsResponseFromHomeProxy
			_, body, err := homeProxyAdapter.Post(ctx, api.TtsPath, &adapter.RequestOptions{
				Headers: map[string]string{"requestId": requestId},
				JSON: TtsRequestToHomeProxy{
					Prompt: req.Text,
				},
			})
			if err != nil {
				logger.Warn("failed to post to home server", zap.Error(err))
				return api.InternalError(c, "cannot login")
			}
			err = json.Unmarshal(body, &baseResponse)
			if err != nil {
				logger.Warn("failed to post to home server", zap.Error(err))
				return api.InternalError(c, "cannot login")
			}

			json.Unmarshal(bybaseResponse.Body, &bodyTtsResponse)

			baseResponse.Body.
				baseResponse.Body["url"]
		}
		//err := insertDailyPlansFunc(ctx, logger)
		//if err != nil {
		//	logger.Error(err.Error(), zap.String("requestId", requestId))
		//	return api.InternalError(c, err.Error())
		//}
		return api.Ok(c, nil)
	}
}
