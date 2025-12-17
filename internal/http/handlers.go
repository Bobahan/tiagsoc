package handlers

import (
	"context"
	"errors"
	"net/http"
	"social/internal/dto"
	"social/internal/entity"
	"social/internal/http/json"
	"social/internal/http/middlewares"
	"social/internal/storage/postgresql"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/k0kubun/pp"
)

const (
	version = "0.0.1"
)

type UsersService interface {
	Get(ctx context.Context, id int64) (*entity.User, error)
}

type PostsService interface {
	Create(ctx context.Context, post *entity.Post) error
	Get(ctx context.Context, id int64) (*entity.Post, error)
	Delete(ctx context.Context, id int64) error
	Update(ctx context.Context, post *entity.Post) error
}

type CommentsService interface {
	GetPostComments(ctx context.Context, id int64) (*entity.Comments, error)
	CreatePostComment(ctx context.Context, comment *entity.Comment) error
}

type HttpHandlers struct {
	userService     UsersService
	postsService    PostsService
	commentsService CommentsService
}

func New(usersService UsersService, postsService PostsService, commentsService CommentsService) http.Handler {
	h := &HttpHandlers{
		userService:     usersService,
		postsService:    postsService,
		commentsService: commentsService,
	}

	mux := mux.NewRouter()
	muxSubrouter := mux.PathPrefix("/v1").Subrouter()

	// !!! Middleware применяются только к v1 !!!
	muxSubrouter.Use(middlewares.LoggingMiddleware)
	muxSubrouter.Use(middlewares.RequestIdMiddleware)

	// Health Router
	muxSubrouter.Path("/health").Methods("GET").HandlerFunc(h.handleHealth)

	// Users Router
	usersRouter := muxSubrouter.PathPrefix("/users").Subrouter()
	usersRouter.Path("/{id}").Methods("GET").HandlerFunc(h.handleGetUser)

	// Posts Router
	postsRouter := muxSubrouter.PathPrefix("/posts").Subrouter()
	postsRouter.Path("/").Methods("POST").HandlerFunc(h.handleCreatePost)
	postsRouter.Path("/{id}").Methods("DELETE").HandlerFunc(h.handleDeletePost)

	postsRouter.Path("/{id}").Methods("GET").Handler(h.FetchPostMiddleware(h.handleGetPost))
	postsRouter.Path("/{id}").Methods("PATCH").Handler(h.FetchPostMiddleware(h.handleUpdatePost))

	return mux
}

func (h *HttpHandlers) handleGetUser(w http.ResponseWriter, r *http.Request) {
	getUuidFromCtx(r)

	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	user, err := h.userService.Get(r.Context(), int64(id))
	if err != nil {
		switch {
		case errors.Is(err, postgresql.ErrorNotFound):
			json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		default:
			json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	if err := json.JsonResponse(w, http.StatusOK, user); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func (h *HttpHandlers) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	getUuidFromCtx(r)

	var payload dto.CreatePostRequest

	if err := json.ReadJson(w, r, &payload); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	// валидация входящих данных
	if err := json.Validate.Struct(payload); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	post := entity.Post{
		Content: payload.Content,
		Title:   payload.Title,
		UserId:  1, // TODO: after auth, it is just mock for now
		Tags:    payload.Tags,
	}

	ctx := r.Context()

	if err := h.postsService.Create(ctx, &post); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := json.JsonResponse(w, http.StatusOK, post); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func (h *HttpHandlers) handleGetPost(w http.ResponseWriter, r *http.Request) {
	getUuidFromCtx(r)
	post := getPostFromCtx(r)

	comments, err := h.commentsService.GetPostComments(r.Context(), post.Id)
	if err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	post.Comments = comments

	if err := json.JsonResponse(w, http.StatusOK, post); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func (h *HttpHandlers) handleDeletePost(w http.ResponseWriter, r *http.Request) {
	getUuidFromCtx(r)
	id, _ := strconv.Atoi(mux.Vars(r)["id"])

	if err := h.postsService.Delete(r.Context(), int64(id)); err != nil {
		switch {
		case errors.Is(err, postgresql.ErrorNotFound):
			json.WriteJsonError(w, http.StatusNotFound, err.Error())
		default:
			json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HttpHandlers) handleUpdatePost(w http.ResponseWriter, r *http.Request) {
	getUuidFromCtx(r)
	post := getPostFromCtx(r)

	var payload dto.UpdatePostRequest

	if err := json.ReadJson(w, r, &payload); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := json.Validate.Struct(payload); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	if payload.Title != nil && *payload.Title != "" {
		post.Title = *payload.Title
	}

	if payload.Content != nil && *payload.Content != "" {
		post.Content = *payload.Content
	}

	// time.Sleep(time.Second * 5) проверка на Race Condition

	if err := h.postsService.Update(r.Context(), post); err != nil {
		switch {
		case errors.Is(err, postgresql.ErrorNotFound):
			json.WriteJsonError(w, http.StatusConflict, err.Error())
		default:
			json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	if err := json.JsonResponse(w, http.StatusOK, post); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

}

func (h *HttpHandlers) handleHealth(w http.ResponseWriter, r *http.Request) {
	getUuidFromCtx(r)
	payload := map[string]string{
		"status":  "ok",
		"version": version,
	}

	if err := json.JsonResponse(w, http.StatusOK, payload); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}
}

// Middleware нужен для GetPost и для UpdatePost
func (h *HttpHandlers) FetchPostMiddleware(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

		if err != nil {
			json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
			return
		}

		ctx := r.Context()

		// mu.Lock()

		// defer mu.Unlock()

		post, err := h.postsService.Get(ctx, id)

		if err != nil {
			switch {
			case errors.Is(err, postgresql.ErrorNotFound):
				json.WriteJsonError(w, http.StatusNotFound, err.Error())
			default:
				json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		ctx = context.WithValue(ctx, middlewares.PostKey, post)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getPostFromCtx(r *http.Request) *entity.Post {
	post, _ := r.Context().Value(middlewares.PostKey).(*entity.Post)

	return post
}

func getUuidFromCtx(r *http.Request) {
	if uuid, ok := r.Context().Value(middlewares.RequestIdKey).(string); ok {
		pp.Printf("Обрабатывается запрос %s ", uuid)
	}
}
