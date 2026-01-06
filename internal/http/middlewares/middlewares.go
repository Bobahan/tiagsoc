package middlewares

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const (
	RequestIdKey contextKey = "requestID"
	PostKey      contextKey = "post"
	UserKey      contextKey = "user"
	UserId       contextKey = "user_id"
)

type MiddlewareManager struct {
	logger *zap.SugaredLogger
}

func New(logger *zap.SugaredLogger) *MiddlewareManager {
	return &MiddlewareManager{logger}
}

func (m *MiddlewareManager) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uuid, _ := r.Context().Value(RequestIdKey).(string)

		m.logger.Info("request", " method: ", r.Method, " requestURI: ", r.RequestURI, " request UUID: ", uuid)
		next.ServeHTTP(w, r)
	})
}

func (m *MiddlewareManager) RequestIdMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uuid := uuid.New().String()

		w.Header().Set("X-REQUEST-ID", uuid)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), RequestIdKey, uuid)))
	})
}
