package auth

//
//import (
//	"github.com/gofiber/fiber/v2"
//	"github.com/jackc/pgx/v5/pgxpool"
//	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
//	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/httputil"
//	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
//	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/utils"
//	"golang.org/x/crypto/bcrypt"
//)
//
//func NewRegisterHandler(
//	dbPool *pgxpool.Pool,
//	sendEmail httputil.HTTPPostRequestFunc,
//) fiber.Handler {
//	return func(c *fiber.Ctx) error {
//		ctx := c.Context()
//		logger := logz.NewLogger()
//		var (
//			req RegisterRequest
//		)
//		if err := c.BodyParser(&req); err != nil {
//			logger.Error(err.Error())
//			return api.BadRequest(c, api.InvalidateBody)
//		}
//		if err := req.Validate(); err != nil {
//			logger.Error(err.Error())
//			return api.BadRequest(c, err.Error())
//		}
//		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
//		if err != nil {
//			return api.InternalError(c, "Error hashing password")
//		}
//		req.Password = string(hashedPassword)
//		sqlRegister := `
//			insert into tbl_users (username, email, name, password, create_by, status)
//				VALUES ($1,$2,$3,$4,$5,$6)
//		`
//		_, err = dbPool.Exec(ctx, sqlRegister, req.UserIdToken, req.Email, req.Name, req.Password, req.UserIdToken, utils.Pending)
//		if err != nil {
//			logger.Error(err.Error())
//			return api.InternalError(c, api.SomeThingWentWrong)
//		}
//		// send mail
//
//		return api.Ok(c, nil)
//	}
//}
