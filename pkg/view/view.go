package view

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/julienschmidt/httprouter"
	"github.com/microcosm-cc/bluemonday"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/env"
	"github.com/Kelvedler/ChemicalStorage/pkg/middleware"
)

type HandlerContext struct {
	dbpool   *pgxpool.Pool
	sanitize *bluemonday.Policy
	validate *validator.Validate
}

func newHandlerContext(
	dbpool *pgxpool.Pool,
	sanitize *bluemonday.Policy,
	validate *validator.Validate,
) *HandlerContext {
	return &HandlerContext{
		dbpool:   dbpool,
		sanitize: sanitize,
		validate: validate,
	}
}

type RequestContext struct {
	logger   *slog.Logger
	userID   string
	dbpool   *pgxpool.Pool
	sanitize *bluemonday.Policy
	validate *validator.Validate
}

type Handle func(*RequestContext, http.ResponseWriter, *http.Request, httprouter.Params)

func baseWrapper(
	handler Handle,
	handlerContext *HandlerContext,
	authRequired bool,
	allowedRoles []db.Role,
) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		logger := middleware.NewRequestLogger()
		middleware.RequestLogger(logger, r)
		if !middleware.HostInAllowed(r, env.Env.AllowedHosts) {
			common.ErrorResp(w, common.Forbidden)
			logger.Info(fmt.Sprintf("Host %s not allowed", r.Host))
			return
		}
		userID, userRole, err := middleware.PerformAuth(
			logger,
			handlerContext.dbpool,
			w,
			r,
			authRequired,
		)
		if err != nil {
			logger.Info(err.Error())
			common.ErrorResp(w, common.Unauthorized)
			return
		}
		if authRequired {
			err = middleware.CheckPermission(userRole, allowedRoles)
			if err != nil {
				logger.Info(err.Error())
				common.ErrorResp(w, common.Forbidden)
				return
			}
		}
		rc := &RequestContext{
			logger:   logger,
			userID:   userID,
			dbpool:   handlerContext.dbpool,
			sanitize: handlerContext.sanitize,
			validate: handlerContext.validate,
		}
		handler(rc, w, r, p)
	}
}

func staticFilepath(mainLogger *slog.Logger) string {
	wd, err := os.Getwd()
	if err != nil {
		mainLogger.Error(err.Error())
		os.Exit(1)
	}
	return wd + "/static/"
}

func BaseRouter(
	dbpool *pgxpool.Pool,
	sanitize *bluemonday.Policy,
	validate *validator.Validate,
	mainLogger *slog.Logger,
) *httprouter.Router {
	router := httprouter.New()
	handlerContext := newHandlerContext(dbpool, sanitize, validate)

	router.GET("/favicon.ico", baseWrapper(Favicon, handlerContext, false, middleware.AllowAll))
	router.ServeFiles("/static/*filepath", http.Dir(staticFilepath(mainLogger)))
	router.GET("/", baseWrapper(Index, handlerContext, false, middleware.AllowAll))
	router.GET("/sign-in", baseWrapper(SignIn, handlerContext, false, middleware.AllowAll))
	router.GET("/sign-up", baseWrapper(SignUp, handlerContext, false, middleware.AllowAll))
	router.GET("/me", baseWrapper(Me, handlerContext, true, middleware.AllowAll))
	router.GET("/users/", baseWrapper(Users, handlerContext, true, middleware.AdminOnly))
	router.GET("/users/:id", baseWrapper(User, handlerContext, true, middleware.AdminOnly))
	router.GET(
		"/reagent-new",
		baseWrapper(ReagentCreate, handlerContext, true, middleware.AssistantOnly),
	)
	router.GET("/reagents/", baseWrapper(Reagents, handlerContext, false, middleware.AllowAll))
	router.GET("/reagents/:id", baseWrapper(Reagent, handlerContext, false, middleware.AllowAll))

	router.POST(
		"/api/v1/sign-in",
		baseWrapper(SignInAPI, handlerContext, false, middleware.AllowAll),
	)
	router.POST(
		"/api/v1/sign-out",
		baseWrapper(SignOutAPI, handlerContext, false, middleware.AllowAll),
	)
	router.POST(
		"/api/v1/sign-up",
		baseWrapper(SignUpAPI, handlerContext, false, middleware.AllowAll),
	)
	router.PUT(
		"/api/v1/users/:id",
		baseWrapper(UserPutAPI, handlerContext, true, middleware.AdminOnly),
	)
	router.GET("/api/v1/users/", baseWrapper(UsersAPI, handlerContext, true, middleware.AdminOnly))
	router.GET(
		"/api/v1/reagents/",
		baseWrapper(ReagentsAPI, handlerContext, false, middleware.AllowAll),
	)
	router.POST(
		"/api/v1/reagents",
		baseWrapper(ReagentCreateAPI, handlerContext, true, middleware.AssistantOnly),
	)
	return router
}
