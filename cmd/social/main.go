package main

import (
	"context"
	"expvar"
	"os"
	"os/signal"
	"runtime"
	"social/internal/app"
	"social/internal/config"
	handlers "social/internal/http"
	"social/internal/mailer"
	"social/internal/ratelimiter"
	comments_service "social/internal/service/comments"
	followers_service "social/internal/service/followers"
	post_service "social/internal/service/posts"
	roles_service "social/internal/service/roles"
	user_service "social/internal/service/users"
	"social/internal/storage"
	"social/internal/storage/postgresql"
	cache "social/internal/storage/redis"
	"syscall"
	"time"

	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// @title						Social App API
// @BasePath					/v1
//
// @securityDefinitions.apiKey	ApiKeyAuth
// @in							header
// @name						Authorization
// @description
func main() {
	config := config.New()

	logger := zap.Must(zap.NewProduction()).Sugar() // sugarizing logger
	defer logger.Sync()

	redis := cache.NewRedisClient(config.Redis.Addr, config.Redis.Password, config.Redis.DB)

	redisStorage := cache.New(redis)

	sql, err := postgresql.New(config)
	if err != nil {
		logger.Fatal("failed to connect to the database: ", err.Error())
	}

	logger.Info("database connection established")
	defer sql.Close()

	storage := storage.New(sql)

	usersService := user_service.New(&storage.Users)
	postsService := post_service.New(&storage.Posts)
	commentsService := comments_service.New(&storage.Comments)
	followersService := followers_service.New(&storage.Followers)
	rolesService := roles_service.New(&storage.Roles)

	mailer := mailer.New(config.Mail.From, config.Mail.ApiKey)

	var store = sessions.NewCookieStore([]byte("MY_SESSION_KEY"))

	rateLimiter := ratelimiter.NewFixedWindowRateLimiter(config.RateLimiter.RequestsPerTimeFrame, config.RateLimiter.TimeFrame)

	handlers := handlers.New(rateLimiter, redisStorage, store, mailer, logger, config, usersService, postsService, commentsService, followersService, rolesService)

	expvar.Publish("database", expvar.Func(func() any {
		return sql.Stats()
	}))

	expvar.Publish("goroutines", expvar.Func(func() any {
		return runtime.NumGoroutine()
	}))

	app := app.New(config, logger, handlers)

	serverErrors := make(chan error, 1)

	go func() {
		serverErrors <- app.Run()
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		logger.Fatal("Server error: ", err.Error())
	case <-signalChan:
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()

		if err := app.Stop(ctx); err != nil {
			logger.Fatal("Graceful shutdown failed: ", err)
		}
	}

	logger.Info("Application shutdown correctly")
}
