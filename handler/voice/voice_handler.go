package voice

import (
	"encoding/json"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/adapter"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
	"go.uber.org/zap"
)

func NewVoceHandler(
	homeProxyAdapter adapter.Adapter,
	updateHitCacheAndReturnAudio UpdateHitCacheAndReturnAudio,
	insertAudioUrlAndKeyToCacheFunc InsertAudioUrlAndKeyToCacheFunc,
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
		cacheKey := utils.BuildCacheKey(req.Text)
		audioUrl, err := updateHitCacheAndReturnAudio(ctx, logger, cacheKey)
		if err != nil || audioUrl == "" {
			logger.Info("audio url not found in cache")
			var ttsResp TtsResponseFromHomeProxy
			_, body, err := homeProxyAdapter.Post(ctx, api.TtsPath, &adapter.RequestOptions{
				Headers: map[string]string{"requestId": requestId},
				JSON: TtsRequestToHomeProxy{
					Prompt: req.Text,
				},
			})
			if err != nil {
				logger.Warn("failed to post to home proxy", zap.Error(err))
				return api.InternalError(c, "cannot get audio url")
			}
			err = json.Unmarshal(body, &ttsResp)
			if err != nil {
				logger.Warn("failed to post to home proxy", zap.Error(err))
				return api.InternalError(c, "cannot get audio url")
			}
			audioUrl = ttsResp.Body.Url

			err = insertAudioUrlAndKeyToCacheFunc(ctx, logger, cacheKey, req.Text, audioUrl, ttsResp.Body.Key)
			if err != nil {
				logger.Warn("failed to post to home proxy", zap.Error(err))
				return api.InternalError(c, "cannot get audio url")
			}
		}
		//err := insertDailyPlansFunc(ctx, logger)
		//if err != nil {
		//	logger.Error(err.Error(), zap.String("requestId", requestId))
		//	return api.InternalError(c, err.Error())
		//}
		// TODO
		return api.Ok(c, fiber.Map{
			"audioUrl": audioUrl,
		})
	}
}
