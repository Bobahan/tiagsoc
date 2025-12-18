package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"social/internal/dto"
	"social/internal/entity"
	"social/internal/http/json"
	"social/internal/http/middlewares"
	"social/internal/service"
	"social/internal/storage/postgresql"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/k0kubun/pp"

	_ "social/docs"

	httpSwagger "github.com/swaggo/http-swagger"
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
	GetUserFeed(ctx context.Context, userId int64, pagination *service.Pagination) (*entity.Feeds, error)
}

type CommentsService interface {
	GetPostComments(ctx context.Context, id int64) (*entity.Comments, error)
	CreatePostComment(ctx context.Context, comment *entity.Comment) error
}

type FollowersService interface {
	FollowUser(ctx context.Context, userId, followerId int64) error
	UnfollowUser(ctx context.Context, userId, followerId int64) error
}

type HttpHandlers struct {
	userService      UsersService
	postsService     PostsService
	commentsService  CommentsService
	followersService FollowersService
}

func New(usersService UsersService, postsService PostsService, commentsService CommentsService, followersService FollowersService) http.Handler {
	h := &HttpHandlers{
		userService:      usersService,
		postsService:     postsService,
		commentsService:  commentsService,
		followersService: followersService,
	}

	mux := mux.NewRouter()

	// init swagger
	docsUrl := fmt.Sprintf("%s/swagger/doc.json", "http://localhost:8080") // по идее нужно получать с local.yaml конфиг файла

	mux.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("%s/swagger/doc.json", docsUrl)),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	)).Methods(http.MethodGet)

	muxSubrouter := mux.PathPrefix("/v1").Subrouter()

	// !!! Middleware применяются только к v1 !!!
	muxSubrouter.Use(middlewares.LoggingMiddleware)
	muxSubrouter.Use(middlewares.RequestIdMiddleware)

	// Health Router
	muxSubrouter.Path("/health").Methods("GET").HandlerFunc(h.handleHealth)

	// Users Router
	usersRouter := muxSubrouter.PathPrefix("/users").Subrouter()

	// userId текущего user будем доставать из сессии
	usersRouter.Path("/feed").Methods("GET").HandlerFunc(h.handleUsersFeed)

	usersRouter.Path("/{id}").Methods("GET").Handler(h.FetchUserMiddleware(h.handleGetUser))
	usersRouter.Path("/{id}/follow").Methods("PUT").Handler(h.FetchUserMiddleware(h.handleFollowUser))
	usersRouter.Path("/{id}/unfollow").Methods("PUT").Handler(h.FetchUserMiddleware(h.handleUnfollowUser))

	// Posts Router
	postsRouter := muxSubrouter.PathPrefix("/posts").Subrouter()
	postsRouter.Path("/").Methods("POST").HandlerFunc(h.handleCreatePost)
	postsRouter.Path("/{id}").Methods("DELETE").HandlerFunc(h.handleDeletePost)

	postsRouter.Path("/{id}").Methods("GET").Handler(h.FetchPostMiddleware(h.handleGetPost))
	postsRouter.Path("/{id}").Methods("PATCH").Handler(h.FetchPostMiddleware(h.handleUpdatePost))

	return mux
}

func (h *HttpHandlers) handleUsersFeed(w http.ResponseWriter, r *http.Request) {
	pagination := service.Pagination{
		Limit:  2,
		Offset: 0,
		Sort:   "desc",
		Search: "",
	}

	parsedPagination, err := pagination.Parse(r)
	if err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := json.Validate.Struct(parsedPagination); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	var userId int64 = 10 // я забираю самого себя

	feed, err := h.postsService.GetUserFeed(r.Context(), userId, parsedPagination)
	if err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := json.JsonResponse(w, http.StatusOK, feed); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

// GetUser godoc
//
//	@Summary		Получить информацию о пользователе
//	@Description	Возвращает данные текущего авторизованного пользователя
//	@Tags			GetUser v1
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int			true	"User Id"
//	@Success		200	{object}	entity.User	"Успешный ответ с объектом пользователя"
//	@Failure		400	{object}	error
//	@Failure		404	{object}	error
//	@Failure		500	{object}	error
//	@Security		ApiKeyAuth
//	@Router			/users/{id} [get]
func (h *HttpHandlers) handleGetUser(w http.ResponseWriter, r *http.Request) {
	getUuidFromCtx(r)

	user := getUserFromCtx(r)

	if err := json.JsonResponse(w, http.StatusOK, user); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func (h *HttpHandlers) handleFollowUser(w http.ResponseWriter, r *http.Request) {
	user := getUserFromCtx(r) // id того user на которого мы хотим подписаться

	// в будущем мы это будем доставать с ctx
	var payload dto.FollowUserRequest // наш user_id

	if err := json.ReadJson(w, r, &payload); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := json.Validate.Struct(payload); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.followersService.FollowUser(r.Context(), payload.UserId, user.Id); err != nil {
		switch {
		case errors.Is(err, postgresql.ErrorDuplicate):
			json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		default:
			json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	json.JsonResponse(w, http.StatusNoContent, nil)
}

func (h *HttpHandlers) handleUnfollowUser(w http.ResponseWriter, r *http.Request) {
	user := getUserFromCtx(r) // id того user на которого мы хотим подписаться

	// в будущем мы это будем доставать с ctx
	var payload dto.FollowUserRequest // наш user_id

	if err := json.ReadJson(w, r, &payload); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := json.Validate.Struct(payload); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.followersService.UnfollowUser(r.Context(), payload.UserId, user.Id); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

// Middleware нужен для GetUser, FollowUser, UnfollowUser
func (h *HttpHandlers) FetchUserMiddleware(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(mux.Vars(r)["id"])

		ctx := r.Context()

		user, err := h.userService.Get(ctx, int64(id))

		if err != nil {
			switch {
			case errors.Is(err, postgresql.ErrorNotFound):
				json.WriteJsonError(w, http.StatusBadRequest, err.Error())
			default:
				json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
			}
			return
		}

		ctx = context.WithValue(ctx, middlewares.UserKey, user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserFromCtx(r *http.Request) *entity.User {
	user, _ := r.Context().Value(middlewares.UserKey).(*entity.User)

	return user
}
