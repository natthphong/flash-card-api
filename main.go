package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	_ "github.com/go-redis/redis/v9"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/google/uuid"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/adapter"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/api"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/config"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/handler/auth"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/handler/daily_plans"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/handler/exam_sessions"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/handler/flashcard_sets"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/handler/job"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/handler/learn"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/handler/voice"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/cache"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/db"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/httputil"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/internal/logz"
	"gitlab.com/home-server7795544/home-server/flash-card/flash-card-api/middleware"
	"go.uber.org/zap"
)

func main() {
	currentTime := time.Now()
	versionDeploy := currentTime.Unix()
	ctx := context.Background()
	app := initFiber()
	config.InitTimeZone()
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.InitConfig()
	if err != nil {
		log.Fatal(errors.New("unable to initial config"))
	}

	logz.Init(cfg.LogConfig.Level, cfg.Server.Name)
	defer logz.Drop()

	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	logger := zap.L()
	logger.Info("version " + strconv.FormatInt(versionDeploy, 10))
	//jsonCfg, err := json.Marshal(cfg)
	//_ = jsonCfg
	//logger.Debug("after cfg : " + string(jsonCfg))
	homeProxyAdapter, _ := adapter.NewAdapter(cfg.HomeProxyAdapter)
	_ = homeProxyAdapter

	homeServerAdapter, _ := adapter.NewAdapter(cfg.HomeServerAdapter)
	_ = homeServerAdapter
	dbPool, err := db.Open(ctx, cfg.DBConfig)
	if err != nil {
		logger.Fatal("server connect to db", zap.Error(err))
	}
	defer dbPool.Close()
	logger.Info("DB CONNECT")

	httpClient := httputil.InitHttpClient(
		cfg.HTTP.TimeOut,
		cfg.HTTP.MaxIdleConn,
		cfg.HTTP.MaxIdleConnPerHost,
		cfg.HTTP.MaxConnPerHost,
	)
	_ = httpClient
	redisClient, err := cache.Initialize(ctx, cfg.RedisConfig)
	if err != nil {
		logger.Fatal("server connect to redis", zap.Error(err))
	}
	redisCMD := redisClient.UniversalClient()
	defer func() {
		err = redisCMD.Close()
		if err != nil {
			logger.Fatal("closing redis connection error", zap.Error(err))
		}
	}()
	logger.Info("Redis Connected")

	app.Use(middleware.AuditLogger())
	app.Use(middleware.JWTMiddleware(cfg.JwtAuthConfig.JwtSecret))
	group := app.Group(fmt.Sprintf("/%s/api/v1", cfg.Server.Name))
	group.Get("/health", func(c *fiber.Ctx) error {
		return api.Ok(c, versionDeploy)
	})

	// auth
	auth.GetRouter(group, *cfg, *homeServerAdapter, dbPool)

	////admin
	//admin.GetRouter(group, *cfg, &redisCMD, dbPool, httputil.NewHttpPostCall(httpClient))
	//flashCardSets
	flashcard_sets.GetRouter(group, *cfg, &redisCMD, dbPool, httputil.NewHttpPostCall(httpClient))

	voice.GetRouter(group, dbPool, *homeProxyAdapter)
	learn.GetRouter(group, dbPool)
	//TODO
	// daily
	daily_plans.GetRouter(group, *cfg, &redisCMD, dbPool, httputil.NewHttpPostCall(httpClient))

	//job
	job.GetRouter(group, *cfg, &redisCMD, dbPool, httputil.NewHttpPostCall(httpClient))

	//exam_sessions
	exam_sessions.GetRouter(group, *cfg, &redisCMD, dbPool, httputil.NewHttpPostCall(httpClient))

	logger.Info(fmt.Sprintf("/%s/api/v1", cfg.Server.Name))
	if err = app.Listen(fmt.Sprintf(":%v", cfg.Server.Port)); err != nil {
		logger.Fatal(err.Error())
	}

}

func initFiber() *fiber.App {
	app := fiber.New(
		fiber.Config{
			BodyLimit:             10 * 1024 * 1024,
			ReadBufferSize:        64 * 1024,
			ReadTimeout:           5 * time.Second,
			WriteTimeout:          5 * time.Second,
			IdleTimeout:           30 * time.Second,
			DisableStartupMessage: true,
			CaseSensitive:         true,
			StrictRouting:         true,
		},
	)
	defaultConfig := cors.ConfigDefault
	defaultConfig.AllowHeaders = "*"
	app.Use(cors.New(defaultConfig))
	app.Use(SetHeaderID())
	return app
}

func SetHeaderID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		randomTrace := uuid.New().String()
		traceId := c.Get("traceId")
		reqId := c.Get("requestId")
		if traceId == "" {
			traceId = randomTrace
		}
		if reqId == "" {
			return api.BadRequest(c, "requestId is required")
		}

		c.Accepts(fiber.MIMEApplicationJSON)
		c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSONCharsetUTF8)
		c.Request().Header.Set("traceId", traceId)
		return c.Next()
	}
}
