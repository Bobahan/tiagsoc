package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"social/internal/config"
	"time"

	_ "github.com/lib/pq"
)

var (
	ErrorNotFound        = errors.New("record not found")
	QueryTimeoutDuration = time.Second * 5
)

func New(config *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", config.Database.Dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(config.Database.MaxOpenConnections)
	db.SetMaxIdleConns(config.Database.MaxIdleConnections)
	db.SetConnMaxIdleTime(config.Database.MaxIdleTime)

	// если запрос идет больше чем 5 сек мы "дропаем" выполнение конкретного запроса
	// "Если выполнение конкретной операции (например, пинг или выборка данных) занимает больше 5 секунд, мы прерываем эту операцию, но сохраняем соединение открытым для будущих запросов."
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}
