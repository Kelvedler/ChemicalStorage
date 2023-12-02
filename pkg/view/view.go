package view

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
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
	userID   uuid.UUID
	userRole db.Role
	dbpool   *pgxpool.Pool
	sanitize *bluemonday.Policy
	validate *validator.Validate
}

type Handle func(*RequestContext, http.ResponseWriter, *http.Request, httprouter.Params)

func baseWrapper(
	handler Handle,
	handlerContext *HandlerContext,
	settings middleware.Settings,
) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		logger := middleware.NewRequestLogger()
		middleware.RequestLogger(logger, r)
		if !middleware.HostInAllowed(r, env.Env.AllowedHosts) {
			common.ErrorResp(w, common.Forbidden)
			logger.Info(fmt.Sprintf("Host %s not allowed", r.Host))
			return
		}
		var userID uuid.UUID
		var userRole db.Role
		var err error
		if !settings.AuthExempt {
			userID, userRole, err = middleware.PerformAuth(
				logger,
				handlerContext.dbpool,
				w,
				r,
				settings.AuthRequired,
			)
			if err != nil {
				logger.Info(err.Error())
				common.ErrorResp(w, common.Unauthorized)
				return
			}

		}
		if !settings.XsrfExempt && !middleware.ValidateForXSRF(r, userID) {
			common.ErrorResp(w, common.Forbidden)
			logger.Warn("XSRF token invalid")
			return
		}
		if settings.AuthRequired {
			err = middleware.CheckPermission(userRole, settings.AllowedRoles)
			if err != nil {
				logger.Info(err.Error())
				common.ErrorResp(w, common.Forbidden)
				return
			}
		}
		rc := &RequestContext{
			logger:   logger,
			userID:   userID,
			userRole: userRole,
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

	router.GET("/favicon.ico", baseWrapper(Favicon, handlerContext, middleware.UnrestrictedNoAuth))
	router.ServeFiles("/static/*filepath", http.Dir(staticFilepath(mainLogger)))
	router.GET("/", baseWrapper(Index, handlerContext, middleware.Unrestricted))
	router.GET("/sign-in", baseWrapper(SignIn, handlerContext, middleware.UnrestrictedNoAuth))
	router.GET("/sign-up", baseWrapper(SignUp, handlerContext, middleware.UnrestrictedNoAuth))
	router.GET("/me", baseWrapper(Me, handlerContext, middleware.Unrestricted))
	router.GET("/users/", baseWrapper(Users, handlerContext, middleware.AdminOnlyView))
	router.GET("/users/:userID", baseWrapper(User, handlerContext, middleware.AdminOnlyView))
	router.GET(
		"/reagent-new",
		baseWrapper(ReagentCreate, handlerContext, middleware.AssistantOnlyView),
	)
	router.GET("/reagents/", baseWrapper(Reagents, handlerContext, middleware.Unrestricted))
	router.GET(
		"/reagents/:reagentID",
		baseWrapper(Reagent, handlerContext, middleware.Unrestricted),
	)
	router.GET(
		"/reagents/:reagentID/instance-new",
		baseWrapper(ReagentInstanceCreate, handlerContext, middleware.AssistantOnlyView),
	)
	router.GET(
		"/storage-new",
		baseWrapper(StorageCreate, handlerContext, middleware.AssistantOnlyView),
	)
	router.GET("/storages/", baseWrapper(Storages, handlerContext, middleware.AssistantOnlyView))
	router.GET(
		"/storages/:storageID",
		baseWrapper(Storage, handlerContext, middleware.AssistantOnlyView),
	)

	router.POST(
		"/api/v1/sign-in",
		baseWrapper(SignInAPI, handlerContext, middleware.Unrestricted),
	)
	router.POST(
		"/api/v1/sign-out",
		baseWrapper(SignOutAPI, handlerContext, middleware.Unrestricted),
	)
	router.POST(
		"/api/v1/sign-up",
		baseWrapper(SignUpAPI, handlerContext, middleware.Unrestricted),
	)
	router.PUT(
		"/api/v1/users/:userID",
		baseWrapper(UserPutAPI, handlerContext, middleware.AdminOnlyAPI),
	)
	router.GET("/api/v1/users/", baseWrapper(UsersAPI, handlerContext, middleware.AdminOnlyAPI))
	router.GET(
		"/api/v1/reagents/",
		baseWrapper(ReagentsAPI, handlerContext, middleware.Unrestricted),
	)
	router.POST(
		"/api/v1/reagents",
		baseWrapper(ReagentCreateAPI, handlerContext, middleware.AssistantOnlyAPI),
	)
	router.PUT(
		"/api/v1/reagents/:reagentID",
		baseWrapper(ReagentPutAPI, handlerContext, middleware.AssistantOnlyAPI),
	)
	router.POST(
		"/api/v1/reagents/:reagentID/instances",
		baseWrapper(ReagentInstanceCreateAPI, handlerContext, middleware.AssistantOnlyAPI),
	)
	router.GET(
		"/api/v1/storages",
		baseWrapper(StoragesAPI, handlerContext, middleware.AssistantOnlyAPI),
	)
	router.POST(
		"/api/v1/storages",
		baseWrapper(StorageCreateAPI, handlerContext, middleware.AssistantOnlyAPI),
	)
	return router
}
