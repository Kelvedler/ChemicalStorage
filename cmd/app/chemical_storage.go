package main

import (
	"context"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/env"
	"github.com/Kelvedler/ChemicalStorage/pkg/view"
)

func main() {
	ctx := context.Background()
	env.InitEnv()
	mainLogger := common.MainLogger()
	validate := validator.New(validator.WithRequiredStructEnabled())
	sanitize := common.GetSanitizer()
	dbpool := db.GetConnectionPool(ctx, mainLogger)
	router := view.BaseRouter(dbpool, sanitize, validate, mainLogger)
	err := http.ListenAndServe(":8000", router)
	mainLogger.Error(err.Error())
}
