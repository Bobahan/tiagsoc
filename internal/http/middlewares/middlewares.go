package middlewares

import (
	"context"
	"log"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	RequestIdKey contextKey = "requestID"
	PostKey      contextKey = "post"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("request %s, %s", r.Method, r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func RequestIdMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uuid := uuid.New().String()

		w.Header().Set("X-REQUEST-ID", uuid)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), RequestIdKey, uuid)))
	})
}
