package auth

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/adapter"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/config"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

const LoginApiPath = "/v1/auth/login"

func LoginHandler(
	homeProxyAdapter adapter.Adapter,
	cfg *config.Config,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req LoginRequest
		var baseResponse api.Response
		var respMap map[string]interface{}
		ctx := c.Context()
		if err := c.BodyParser(&req); err != nil {
			return api.BadRequest(c, "Invalid input")
		}
		reqToHomeServer := LoginRequestToHomeServer{
			UserId:      req.UserId,
			Password:    req.Password,
			AppCode:     cfg.AppCode,
			CompanyCode: cfg.CompanyCode,
		}
		reqId := uuid.New().String()

		logger := logz.NewLogger()
		logger.Info("request to home server", zap.String("reqId", reqId))
		_, body, err := homeProxyAdapter.Post(ctx, LoginApiPath, &adapter.RequestOptions{
			Headers: map[string]string{"requestId": reqId},
			JSON:    reqToHomeServer,
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
		respMap = baseResponse.Body.(map[string]interface{})
		token, ok := respMap["userIdToken"]
		if !ok {
			return api.BadRequest(c, "userIdToken not found")
		}

		tokenStr, ok := token.(string)
		if !ok || tokenStr == "" {
			return api.BadRequest(c, "invalid userIdToken")
		}
		delete(respMap, "userIdToken")
		logger.Info("response grom home server", zap.String("tokenStr", tokenStr))
		// TODO  insert and merge to table
		return api.OkFromResponse(c, respMap)
	}
}
