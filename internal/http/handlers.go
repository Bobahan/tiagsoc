package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"social/internal/config"
	"social/internal/dto"
	"social/internal/entity"
	"social/internal/http/json"
	"social/internal/http/jwt"
	"social/internal/http/middlewares"
	"social/internal/mailer"
	"social/internal/ratelimiter"
	"social/internal/service"
	"social/internal/storage/postgresql"
	cache "social/internal/storage/redis"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/k0kubun/pp"
	"go.uber.org/zap"

	_ "social/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	version = "0.0.1"
)

type UsersService interface {
	GetUser(ctx context.Context, id int64) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	RegisterUser(ctx context.Context, user *entity.User, token string, exp time.Duration) error
	ActivateUser(ctx context.Context, token string) error
	Delete(ctx context.Context, userId int64) error
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

type RolesService interface {
	GetByName(ctx context.Context, roleName string) (*entity.Role, error)
}

type HttpHandlers struct {
	rateLimiter      ratelimiter.Limiter
	redis            cache.Storage
	store            *sessions.CookieStore
	mailer           mailer.Client
	logger           *zap.SugaredLogger
	config           *config.Config
	userService      UsersService
	postsService     PostsService
	commentsService  CommentsService
	followersService FollowersService
	rolesService     RolesService
}

func New(
	rateLimiter ratelimiter.Limiter,
	redis cache.Storage,
	store *sessions.CookieStore,
	mailer mailer.Client,
	logger *zap.SugaredLogger,
	config *config.Config,
	usersService UsersService,
	postsService PostsService,
	commentsService CommentsService,
	followersService FollowersService,
	rolesService RolesService,

) http.Handler {
	h := &HttpHandlers{
		rateLimiter:      rateLimiter,
		redis:            redis,
		store:            store,
		mailer:           mailer,
		logger:           logger,
		config:           config,
		userService:      usersService,
		postsService:     postsService,
		commentsService:  commentsService,
		followersService: followersService,
		rolesService:     rolesService,
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
	middlewares := middlewares.New(h.logger)

	muxSubrouter.Use(middlewares.RequestIdMiddleware)
	muxSubrouter.Use(middlewares.LoggingMiddleware)

	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:3000"
	}

	handlers.CORS(
		handlers.AllowedOrigins([]string{allowedOrigin}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"}),
		handlers.AllowCredentials(),
		handlers.MaxAge(300),
	)

	muxSubrouter.Use(h.RateLimiterMiddleware)

	// Health Router
	muxSubrouter.Path("/health").Methods("GET").HandlerFunc(h.handleHealth)

	usersPublicRouter := muxSubrouter.PathPrefix("/users").Subrouter()
	usersPublicRouter.Path("/login").Methods("POST").HandlerFunc(h.handleLoginUser)

	usersPrivateRouter := usersPublicRouter.PathPrefix("").Subrouter()
	usersPrivateRouter.Use(h.AuthMiddleware)
	usersPrivateRouter.Path("/feed").Methods("GET").HandlerFunc(h.handleUsersFeed)
	usersPrivateRouter.Path("/{id}").Methods("GET").Handler(h.FetchUserMiddleware(h.handleGetUser))
	usersPrivateRouter.Path("/{id}/follow").Methods("PUT").Handler(h.FetchUserMiddleware(h.handleFollowUser))
	usersPrivateRouter.Path("/{id}/unfollow").Methods("PUT").Handler(h.FetchUserMiddleware(h.handleUnfollowUser))

	// Posts Router
	postsRouter := muxSubrouter.PathPrefix("/posts").Subrouter()
	postsRouter.Use(h.AuthMiddleware)
	postsRouter.Use(h.FetchPostMiddleware)
	postsRouter.Path("").Methods("POST").HandlerFunc(h.handleCreatePost)
	postsRouter.Path("/{id}").Methods("DELETE").Handler(h.CheckPostPermissionMiddleware("admin", h.handleDeletePost))
	postsRouter.Path("/{id}").Methods("GET").HandlerFunc(h.handleGetPost)
	postsRouter.Path("/{id}").Methods("PATCH").Handler(h.CheckPostPermissionMiddleware("moderator", h.handleUpdatePost))

	// Регистрация пользователя, public routes
	authRouter := muxSubrouter.PathPrefix("/authentication").Subrouter()
	authRouter.Path("/user").Methods("POST").HandlerFunc(h.handleRegisterUser)
	authRouter.Path("/activate/{token}").Methods("PUT").HandlerFunc(h.handleActivateUser)

	return mux
}

func (h *HttpHandlers) handleLoginUser(w http.ResponseWriter, r *http.Request) {
	var payload dto.LoginUserPayload

	if err := json.ReadJson(w, r, &payload); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := json.Validate.Struct(payload); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.userService.GetByEmail(r.Context(), payload.Email)
	if err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := user.Password.Compare(payload.Password); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	token, err := jwt.CreateToken(user.Id)
	if err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 1. Получаем сессию (создается новая, если её нет)
	session, _ := h.store.Get(r, "auth-session")

	// 2. Устанавливаем настройки безопасности
	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   3600, // время жизни в секундах (1 час)
		HttpOnly: true, // защита от XSS
		Secure:   true, // только для HTTPS
		SameSite: http.SameSiteLaxMode,
	}

	// 3. Записываем токен в сессию
	session.Values["access_token"] = token

	// 4.Сохраняем сессию в куки
	if err := session.Save(r, w); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *HttpHandlers) handleActivateUser(w http.ResponseWriter, r *http.Request) {
	// ourToken := "582e9648-e82e-4d15-866d-490da4e78383"

	token := mux.Vars(r)["token"]

	if err := h.userService.ActivateUser(r.Context(), token); err != nil {
		switch {
		case errors.Is(err, postgresql.ErrorNotFound):
			json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		default:
			json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		}

		return
	}

	if err := json.JsonResponse(w, http.StatusNoContent, nil); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func (h *HttpHandlers) handleRegisterUser(w http.ResponseWriter, r *http.Request) {
	var payload dto.RegisterUserPayload

	if err := json.ReadJson(w, r, &payload); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := json.Validate.Struct(payload); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	user := &entity.User{
		Username: payload.Username,
		Email:    payload.Email,
	}

	if err := user.Password.Set(payload.Password); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	uuid := uuid.New().String()

	if err := h.userService.RegisterUser(r.Context(), user, uuid, h.config.MailInvitationExpired); err != nil {
		// не получилось зарегистрироваться
		switch {
		case errors.Is(err, postgresql.ErrorDuplicateEmail):
			json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, postgresql.ErrorDuplicateUsername):
			json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		default:
			json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		}

		return
	}

	userWithToken := dto.UserWithToken{
		User:  user,
		Token: uuid,
	}

	activationUrl := fmt.Sprintf("%s/confirm/%s", h.config.FrontUrl, userWithToken.Token)

	isProdEnv := h.config.Env == "production"

	vars := struct {
		Username   string
		Activation string
	}{
		Username:   user.Username,
		Activation: activationUrl,
	}

	if err := h.mailer.Send(mailer.UserWelcomeTemplate, user.Username, user.Email, vars, !isProdEnv); err != nil {
		h.logger.Errorw("error sending welcome email", "error", err)

		// rollback user creation if emails fails SAGA PATTERN
		if err := h.userService.Delete(r.Context(), user.Id); err != nil {
			h.logger.Errorw("error deleting error", "error", err)
		}

		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := json.JsonResponse(w, http.StatusCreated, userWithToken); err != nil {
		json.WriteJsonError(w, http.StatusBadRequest, err.Error())
		return
	}
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

	var user = r.Context().Value(middlewares.UserKey).(*entity.User) // я забираю самого себя

	feed, err := h.postsService.GetUserFeed(r.Context(), user.Id, parsedPagination)
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
	user := getUserFromCtx(r)

	if err := json.JsonResponse(w, http.StatusOK, user); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

func (h *HttpHandlers) handleFollowUser(w http.ResponseWriter, r *http.Request) {
	userToFollow := getUserFromCtx(r) // id того user на которого мы хотим подписаться

	var user = r.Context().Value(middlewares.UserKey).(*entity.User) // я забираю самого себя, наш user_id

	if err := h.followersService.FollowUser(r.Context(), user.Id, userToFollow.Id); err != nil {
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
	userToUnfollow := getUserFromCtx(r) // id того user с которого мы хотим отписаться

	var user = r.Context().Value(middlewares.UserKey).(*entity.User) // я забираю самого себя,  наш user_id

	if err := h.followersService.UnfollowUser(r.Context(), user.Id, userToUnfollow.Id); err != nil {
		json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *HttpHandlers) handleCreatePost(w http.ResponseWriter, r *http.Request) {
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

	var user = r.Context().Value(middlewares.UserKey).(*entity.User) // я забираю самого себя

	post := entity.Post{
		Content: payload.Content,
		Title:   payload.Title,
		UserId:  user.Id,
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

	// var user = r.Context().Value(middlewares.UserKey).(*entity.User) // я забираю самого себя,  наш user_id

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
func (h *HttpHandlers) FetchPostMiddleware(next http.Handler) http.Handler {
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

// Middleware нужен для GetUser, FollowUser, UnfollowUser
func (h *HttpHandlers) FetchUserMiddleware(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(mux.Vars(r)["id"])

		ctx := r.Context()

		user, err := h.userService.GetUser(ctx, int64(id))

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

func (h *HttpHandlers) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Достаем сессию
		session, _ := h.store.Get(r, "auth-session")

		// Из сессии достаем access_token
		token, ok := session.Values["access_token"].(string)
		if !ok || token == "" {
			json.WriteJsonError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// Валидация токена
		claims, err := jwt.ExtractClaims(token)
		if err != nil {
			json.WriteJsonError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		// После валидации достаем id пользователя который мы туда положили когда логинились
		id, ok := claims["id"]
		if !ok {
			json.WriteJsonError(w, http.StatusUnauthorized, "Invalid token data")
			return
		}

		// Конвертируем: сначала в float64 (как хранит JWT), затем в int64
		floatId, ok := id.(float64)
		if !ok {
			json.WriteJsonError(w, http.StatusUnauthorized, "ID is not a number")
			return
		}

		userId := int64(floatId)

		user, err := h.getUser(r.Context(), userId)
		if err != nil {
			json.WriteJsonError(w, http.StatusInternalServerError, err.Error())
			return
		}

		pp.Println(user)

		// Теперь кладем в контекст уже готовый int64
		ctx := context.WithValue(r.Context(), middlewares.UserKey, user)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Мы должны быть залогинены прежде чем вызвать этот middleware
func (h *HttpHandlers) CheckPostPermissionMiddleware(role string, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := getUserFromCtx(r)
		post := getPostFromCtx(r)

		// проверка на то является пользователь автором
		if post.UserId == user.Id {
			next.ServeHTTP(w, r)
			return
		}

		isAllowed, err := h.checkRolePermission(r.Context(), user, role)
		if err != nil {
			json.WriteJsonError(w, http.StatusForbidden, err.Error())
			return
		}

		if !isAllowed {
			json.WriteJsonError(w, http.StatusForbidden, "Forbidden")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *HttpHandlers) checkRolePermission(ctx context.Context, user *entity.User, roleName string) (bool, error) {
	role, err := h.rolesService.GetByName(ctx, roleName)
	if err != nil {
		return false, err
	}

	return user.Role.Level >= role.Level, nil
}

func (h *HttpHandlers) getUser(ctx context.Context, userId int64) (*entity.User, error) {
	user, err := h.redis.Get(ctx, userId)
	if err != nil {
		return nil, err
	}

	if user == nil { // не смогли достать пользователя из кэша
		// идем доставать из PostgreqSQL
		user, err := h.userService.GetUser(ctx, userId) // Достаем user и кладем его в контекст, чтобы в других местах доставать его
		if err != nil {
			return nil, err
		}

		if err := h.redis.Set(ctx, user); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (h *HttpHandlers) RateLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.config.RateLimiter.Enabled {
			ip := r.RemoteAddr

			if allow, retryAfter := h.rateLimiter.Allow(ip); !allow {
				json.RateLimitExceededResponse(w, r, retryAfter.String())
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}
