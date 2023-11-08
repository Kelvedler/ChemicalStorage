package view

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/julienschmidt/httprouter"
	"github.com/microcosm-cc/bluemonday"

	"github.com/Kelvedler/ChemicalStorage/pkg/auth"
	"github.com/Kelvedler/ChemicalStorage/pkg/common"
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

func writeForbidden(w http.ResponseWriter, logger *slog.Logger) {
	errMap := make(map[string]string)
	errMap["error"] = "Forbidden"
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(403)
	errByte, err := json.Marshal(errMap)
	if err != nil {
		logger.Error(err.Error())
	} else {
		w.Write(errByte)
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
	allowedRoles []string,
) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		logger := middleware.NewRequestLogger()
		middleware.RequestLogger(logger, r)
		if !middleware.HostInAllowed(r, env.Env.AllowedHosts) {
			writeForbidden(w, logger)
			logger.Info(fmt.Sprintf("Host %s not allowed", r.Host))
			return
		}
		userID, err := middleware.PerformAuth(
			logger,
			handlerContext.dbpool,
			w,
			r,
			authRequired,
			allowedRoles,
		)
		if err != nil {
			logger.Info(err.Error())
			common.ErrorRespUnauthorized(w)
			return
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

	router.GET("/favicon.ico", baseWrapper(Favicon, handlerContext, false, auth.AllowAll))
	router.ServeFiles("/static/*filepath", http.Dir(staticFilepath(mainLogger)))
	router.GET("/", baseWrapper(Index, handlerContext, false, auth.AllowAll))
	router.GET("/sign-in", baseWrapper(SignIn, handlerContext, false, auth.AllowAll))
	router.GET("/sign-up", baseWrapper(SignUp, handlerContext, false, auth.AllowAll))
	router.GET("/me", baseWrapper(Me, handlerContext, true, auth.AllowAll))
	router.GET("/reagent-new", baseWrapper(ReagentCreate, handlerContext, false, auth.AllowAll))
	router.GET("/reagents/", baseWrapper(Reagents, handlerContext, false, auth.AllowAll))
	router.GET("/reagents/:id", baseWrapper(Reagent, handlerContext, false, auth.AllowAll))

	router.POST("/api/v1/sign-in", baseWrapper(SignInAPI, handlerContext, false, auth.AllowAll))
	router.POST("/api/v1/sign-out", baseWrapper(SignOutAPI, handlerContext, false, auth.AllowAll))
	router.POST("/api/v1/sign-up", baseWrapper(SignUpAPI, handlerContext, false, auth.AllowAll))
	router.GET("/api/v1/reagents/", baseWrapper(ReagentsAPI, handlerContext, false, auth.AllowAll))
	router.POST(
		"/api/v1/reagents",
		baseWrapper(ReagentCreateAPI, handlerContext, false, auth.AllowAll),
	)
	return router
}
