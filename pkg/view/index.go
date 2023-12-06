package view

import (
	"html/template"
	"net/http"

	"github.com/julienschmidt/httprouter"

	"github.com/Kelvedler/ChemicalStorage/pkg/db"
	"github.com/Kelvedler/ChemicalStorage/pkg/middleware"
)

type CallerData struct {
	Caller db.StorageUser
}

func Index(
	rc *middleware.RequestContext,
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	caller := db.StorageUser{ID: rc.UserID}
	_ = db.PerformBatch(r.Context(), rc.DBpool, []db.BatchSet{caller.GetByID})
	data := CallerData{
		Caller: caller,
	}
	tmpl := template.Must(template.ParseFiles("templates/index.html", "templates/base.html"))
	tmpl.Execute(w, data)
}
