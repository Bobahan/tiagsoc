package internal_error

import (
	"log"
	"net/http"
	"social/internal/http/json"
)

func InternalServerError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("internal server error, error: %s", err.Error())
	json.WriteJsonError(w, http.StatusInternalServerError, "server has a problem")
}

func BadRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("bad request error, error: %s", err)
	json.WriteJsonError(w, http.StatusBadRequest, err.Error())
}

func NotFoundResponse(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("not found error, error: %s", err)
	json.WriteJsonError(w, http.StatusNotFound, "not found")
}
