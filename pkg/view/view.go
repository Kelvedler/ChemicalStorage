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
	logger *slog.Logger
}

type Handle func(RequestContext, http.ResponseWriter, *http.Request, httprouter.Params)

func handlerWrapper(handler Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		logger := middleware.NewRequestLogger()
		middleware.RequestLogger(logger, r)
		if !middleware.HostInAllowed(r, os.Getenv("ALLOWED_HOSTS")) {
			writeForbidden(w, logger)
			logger.Info(fmt.Sprintf("Host %s not allowed", r.Host))
			return
		}
		rc := RequestContext{logger: logger}
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

	router.GET("/favicon.ico", handlerWrapper(handlerContext.Favicon))
	router.ServeFiles("/static/*filepath", http.Dir(staticFilepath(mainLogger)))
	router.GET("/", handlerWrapper(handlerContext.Index))
	router.GET("/reagent-new", handlerWrapper(handlerContext.ReagentCreate))
	router.GET("/reagents/", handlerWrapper(handlerContext.Reagents))
	router.GET("/reagents/:id", handlerWrapper(handlerContext.Reagent))
	router.GET("/api/v1/reagents/", handlerWrapper(handlerContext.ReagentsAPI))
	router.POST("/api/v1/reagents", handlerWrapper(handlerContext.ReagentCreateAPI))
	return router
}
