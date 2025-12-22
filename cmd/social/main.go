package main

import (
	"context"
	"os"
	"os/signal"
	"social/internal/app"
	"social/internal/config"
	handlers "social/internal/http"
	"social/internal/mailer"
	comments_service "social/internal/service/comments"
	followers_service "social/internal/service/followers"
	post_service "social/internal/service/posts"
	user_service "social/internal/service/users"
	"social/internal/storage"
	"social/internal/storage/postgresql"
	"syscall"
	"time"

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

	mailer := mailer.New(config.Mail.From, config.Mail.ApiKey)

	handlers := handlers.New(mailer, logger, config, usersService, postsService, commentsService, followersService)

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
