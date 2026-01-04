package admin

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
	"go.uber.org/zap"
)

func NewConfirmUserHandler(db *pgxpool.Pool) fiber.Handler {

	return func(c *fiber.Ctx) error {
		var (
			req ConfirmUserRequest
		)
		logger := logz.NewLogger()
		requestId := c.Get("requestId")

		if err := c.BodyParser(&req); err != nil {
			return api.BadRequest(c, api.InvalidateBody)
		}
		if err := req.Validate(); err != nil {
			return api.BadRequest(c, err.Error())
		}
		// check first

		updateSql := `
				update tbl_users set status = $1 , update_at = now() , update_by = 'ADMIN'
                 where username= $2
			`
		_, err := db.Exec(c.Context(), updateSql, utils.APPROVED, req.Username)
		if err != nil {
			logger.Error(err.Error(), zap.String("requestId", requestId))
			return api.InternalError(c, api.SomeThingWentWrong)
		}
		// send mail

		return api.Ok(c, nil)
	}
}
