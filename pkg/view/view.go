package view

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/julienschmidt/httprouter"
)

type BaseHandler struct {
	dbpool   *pgxpool.Pool
	validate *validator.Validate
}

func newBaseHandler(
	dbpool *pgxpool.Pool,
	validate *validator.Validate,
) *BaseHandler {
	return &BaseHandler{
		dbpool:   dbpool,
		validate: validate,
	}
}

func staticFilepath() string {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return wd + "/static/"
}

func BaseRouter(
	dbpool *pgxpool.Pool,
	validate *validator.Validate,
) *httprouter.Router {
	router := httprouter.New()
	handler := newBaseHandler(dbpool, validate)
	router.GET("/favicon.ico", handler.Favicon)
	router.ServeFiles("/static/*filepath", http.Dir(staticFilepath()))
	router.GET("/", handler.Index)
	router.GET("/reagent-new", handler.ReagentCreate)
	router.GET("/reagents/", handler.Reagents)
	router.GET("/reagents/:id", handler.Reagent)
	router.GET("/api/v1/reagents/", handler.ReagentsAPI)
	router.POST("/api/v1/reagents", handler.ReagentCreateAPI)
	return router
}