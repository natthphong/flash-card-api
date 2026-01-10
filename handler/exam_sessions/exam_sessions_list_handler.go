package exam_sessions

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
	"go.uber.org/zap"
)

func NewExamSessionsListHandler(
	listExamHistoryFunc ListExamHistoryFunc,
) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req ExamSessionListRequest
		ctx := c.Context()
		logger := logz.NewLogger()
		requestId := c.Get("requestId")

		if err := c.BodyParser(&req); err != nil {
			logger.Error("body parse error", zap.String("requestId", requestId), zap.Error(err))
			return api.BadRequest(c, api.InvalidateBody)
		}
		validate := validator.New()
		if err := validate.Struct(req); err != nil {
			return api.BadRequest(c, api.InvalidateBody)
		}
		userIdToken := utils.GetUserIDToken(c)
		resp, err := listExamHistoryFunc(ctx, logger, req.Page, req.Size, userIdToken, req.SearchBy)
		if err != nil {
			logger.Error("list exam sessions list error", zap.String("requestId", requestId), zap.Error(err))
			return api.InternalError(c, err.Error())
		}
		return api.Ok(c, resp)
	}
}
