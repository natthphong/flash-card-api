package auth

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/adapter"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/config"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"go.uber.org/zap"
)

const LoginApiPath = "/v1/auth/login"

// TODO
func LoginHandler(
	homeProxyAdapter adapter.Adapter,
	cfg *config.Config,
	dbpool *pgxpool.Pool,
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
		//delete(respMap, "userIdToken")
		logger.Info("response grom home server", zap.String("tokenStr", tokenStr))
		sql := `
			insert into tbl_user_config (user_id_token, status, daily_active, daily_target)
			values ($1,'ACTIVE','N',20)
			on conflict(user_id_token) do nothing ;
			`
		_, err = dbpool.Exec(ctx, sql, tokenStr)
		if err != nil {
			logger.Warn("failed to insert into tbl_user_config", zap.Error(err))
			return api.InternalError(c, "DATABASE error")
		}
		return api.OkFromResponse(c, respMap)
	}
}
