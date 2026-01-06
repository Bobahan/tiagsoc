package json

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/k0kubun/pp"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New(validator.WithRequiredStructEnabled())
}

func WriteJson(w http.ResponseWriter, status int, payload interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	return json.NewEncoder(w).Encode(&payload)
}

func ReadJson(w http.ResponseWriter, r *http.Request, payload interface{}) error {
	// ограничивем размер запроса, чтобы не было DDoS атак
	maxBytes := 1_048_576 // 1 МБ
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	decoder := json.NewDecoder(r.Body)

	// если у нас есть unknown fields, которые не метчатся с payload, то будет error
	decoder.DisallowUnknownFields()

	return decoder.Decode(payload)
}

// Универсальная функция ответа ошибки для клиента
func WriteJsonError(w http.ResponseWriter, status int, message string) error {
	type errorDto struct {
		Error string `json:"error"`
	}

	return WriteJson(w, status, &errorDto{Error: message})
}

func JsonResponse(w http.ResponseWriter, status int, data interface{}) error {
	type dto struct {
		Data interface{} `json:"data"`
	}

	return WriteJson(w, status, &dto{Data: data})
}

func RateLimitExceededResponse(w http.ResponseWriter, r *http.Request, retryAfter string) {
	pp.Println("rate limit exceeded", "method", r.Method, "path", r.URL.Path)
	w.Header().Set("Retry-After", retryAfter)
	WriteJsonError(w, http.StatusTooManyRequests, fmt.Sprintf("rate limit exceeded, try after: %s", retryAfter))
}
