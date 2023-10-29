package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/Kelvedler/ChemicalStorage/pkg/common"
	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/view"
)

func main() {
	ctx := context.Background()
	common.LoadDotenv()
	validate := validator.New(validator.WithRequiredStructEnabled())
	sanitize := common.GetSanitizer()
	dbpool := db.GetConnectionPool(ctx)
	router := view.BaseRouter(dbpool, sanitize, validate)
	log.Fatal(http.ListenAndServe(":8000", router))
}
